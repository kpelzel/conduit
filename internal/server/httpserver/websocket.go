// Copyright 2026. Triad National Security, LLC. All rights reserved.

package httpserver

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lanl/conduit/api"
	"github.com/lanl/conduit/internal/logger"
	"google.golang.org/protobuf/proto"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Client is a middleman between the websocket connection and the hub.
type WebsocketClient struct {
	log *logger.ConduitLogger
	// The websocket connection.
	conn *websocket.Conn

	// the grpc client to conduit
	conduitClient api.ConduitApiClient

	// Buffered channel of outbound messages.
	send chan []byte
}

// serveWs handles websocket requests from the peer.
func (h *HTTPServer) serveWs(wr http.ResponseWriter, req *http.Request, username string) {
	// double check the method isn't HEAD
	if req.Method != http.MethodGet {
		http.Error(wr, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.originPolicy.CheckWebsocketOrigin,
	}

	conn, err := upgrader.Upgrade(wr, req, nil)
	if err != nil {
		h.log.Errorf("failed to upgrade websocket connection: %v", err)
		return
	}

	client := &WebsocketClient{
		conn:          conn,
		send:          make(chan []byte, 256),
		log:           h.log,
		conduitClient: h.conduitClient,
	}

	nctx, nCancel := context.WithCancelCause(req.Context())

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump(nctx, nCancel)
	go client.readPump(nctx, nCancel)

	err = client.monitorTransfers(nctx, username)
	if err != nil {
		nCancel(err)
		h.log.Error(err)
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *WebsocketClient) writePump(ctx context.Context, cancel context.CancelCauseFunc) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		cancel(nil)
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				err = fmt.Errorf("error writing to websocket connection: %w", err)
				c.log.Error(err)
				cancel(err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				cancel(fmt.Errorf("error writing websocket ping: %w", err))
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *WebsocketClient) readPump(ctx context.Context, cancel context.CancelCauseFunc) {
	defer c.conn.Close()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				cancel(fmt.Errorf("websocket unexpectedly closed: %w", err))
			} else {
				cancel(nil)
			}
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

// monitorTransfers is a long running function that will stream transfer updates from conduit to the websocket connection
func (c *WebsocketClient) monitorTransfers(ctx context.Context, username string) error {
	// setup a notify channel to check if new transfers come in for a user
	// notifyTransfers will also setup the watch channel with conduit and start returning updates for any new transfers for a user
	// TODO: watch for this context to get cancelled?
	nctx, nCancel := context.WithCancelCause(ctx)
	tChan := make(chan map[string]bool, 1)
	retChan := make(chan *api.MultiTransferDetails, 100)
	go c.notifyTransfers(nctx, nCancel, username, tChan, retChan)

	// query conduit and send all current transfers for a user to the tChan
	mtd, err := c.conduitClient.Query(ctx, &api.QueryOptions{User: username})
	if err != nil {
		tErr := fmt.Errorf("failed to query conduit for transfers: %v", err)
		nCancel(tErr)
		return tErr
	}

	transfers := make(map[string]bool)

	for _, t := range mtd.GetDetails() {
		if t.GetActive() {
			transfers[t.GetTransferID()] = true
		}
	}

	select {
	case tChan <- transfers:
	case <-ctx.Done():
		return ctx.Err()
	}

	mtdBytes, err := proto.Marshal(mtd)
	if err != nil {
		return fmt.Errorf("failed to marshal multi transfer details: %v", err)
	}

	select {
	case c.send <- mtdBytes:
	case <-ctx.Done():
		return ctx.Err()
	}

	for {
		select {
		case mtd = <-retChan:
			mtdBytes, err := proto.Marshal(mtd)
			if err != nil {
				return fmt.Errorf("failed to marshal multi transfer details: %v", err)
			}

			select {
			case c.send <- mtdBytes:
			case <-ctx.Done():
				return ctx.Err()
			}
		case <-ctx.Done():
			return fmt.Errorf("websocket context done")
		case <-nctx.Done():
			return fmt.Errorf("notify context done: %v", context.Cause(nctx))
		}
	}

}

func (c *WebsocketClient) notifyTransfers(ctx context.Context, cancel context.CancelCauseFunc, username string, tChan chan map[string]bool, retChan chan *api.MultiTransferDetails) {
	// get notify channel from conduit
	nc, err := c.conduitClient.TransferNotify(ctx, &api.NotifyRequest{
		User: username,
	})
	// If watch request not successful, err
	if err != nil {
		cancel(fmt.Errorf("failed to get transfer notifications: %v", err))
		return
	}
	go func() {
		<-nc.Context().Done()
		c.log.Debug("transfer notify: grpc stream closed")
	}()
	// Close the send side of grpc stream since we are only receiving
	err = nc.CloseSend()
	// Err if could not close send side of grpc stream
	if err != nil {
		cancel(fmt.Errorf("failed to close send channel of transfer notifications: %v", err))
		return
	}

	wchan := make(chan *api.MultiTransferDetails)

	// a routine to pass any updates from the watch to the parent channel
	go func() {
		for {
			select {
			case mtd := <-wchan:
				select {
				case retChan <- mtd:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				// if the parent context dies, kill this go routine
				return
			}
		}
	}()

	nchan := make(chan *api.NotifyMessage, 100)
	nctx, nCancel := context.WithCancelCause(ctx)

	// a routine to receive from the notify stream
	go func() {
		for {
			gnm, err := nc.Recv()
			if err != nil {
				nCancel(fmt.Errorf("error while receiving notify message: %w", err))
				return
			}

			select {
			case nchan <- gnm:
			case <-nctx.Done():
				return
			}
		}
	}()

	var transfers map[string]bool

	// wait to get something from the query before we start sending updates to the websocket
	// or return if the context is already dead
	select {
	case transfers = <-tChan:
	case <-nctx.Done():
		cancel(context.Cause(nctx))
		return
	case <-ctx.Done():
		return
	}

	// start a grpc watch every time we get a notification that a transfer has been added or deleted
	var nm *api.NotifyMessage
	for {

		wctx, wCancel := context.WithCancelCause(ctx)
		if len(transfers) > 0 {
			go c.watchTransfers(wctx, wCancel, username, maps.Clone(transfers), wchan)
		}

		select {
		case nm = <-nchan:
			// close the current watch channels so we can start another one
			wCancel(fmt.Errorf("notify closing watch channel"))

			if nm.GetCreated() {
				transfers[nm.GetTransferID()] = true
			} else {
				delete(transfers, nm.GetTransferID())
			}
		case <-nctx.Done():
			wCancel(context.Cause(nctx))
			cancel(context.Cause(nctx))
			return
		case <-wctx.Done():
			if ctx.Err() != nil {
				wCancel(nil)
				return
			}

			cause := context.Cause(wctx)
			c.log.Warnf("watch transfers failed: %v", cause)

			timer := time.NewTimer(time.Second)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				wCancel(nil)
				return
			}
		}
	}
}

func (c *WebsocketClient) watchTransfers(ctx context.Context, cancel context.CancelCauseFunc, username string, transfers map[string]bool, ch chan *api.MultiTransferDetails) {
	transferSlice := make([]string, 0, len(transfers))
	for k := range transfers {
		transferSlice = append(transferSlice, k)
	}

	// get watch channel from conduit
	wc, err := c.conduitClient.WatchStatus(ctx, &api.TransferIds{
		User:  username,
		Value: transferSlice,
	})
	// If watch request not successful, err
	if err != nil {
		cancel(fmt.Errorf("failed to watch transfers in conduit: %v", err))
		return
	}

	go func() {
		<-wc.Context().Done()
		c.log.Debug("transfer watch: grpc stream closed")
	}()
	// Close the send side of grpc stream since we are only receiving
	err = wc.CloseSend()
	// Err if could not close send side of grpc stream
	if err != nil {
		cancel(fmt.Errorf("failed to close send channel of transfer watch: %v", err))
		return
	}

	var mtd *api.MultiTransferDetails

	for {
		// Receive status
		mtd, err = wc.Recv()
		// If error in receiving status, err
		if err != nil {
			cancel(fmt.Errorf("watch: error while watching status: %v", err))
			return
		}
		// Get slice of TransferDetails from map of TransferDetails
		cmtd := proto.Clone(mtd).(*api.MultiTransferDetails)

		select {
		case ch <- cmtd:
		case <-ctx.Done():
			return
		}
	}

}

func newWebsocketCheckOrigin(log *logger.ConduitLogger, allowedOrigins []string) func(*http.Request) bool {
	allowed := make(map[string]struct{})

	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}

		u, err := url.Parse(origin)
		if err != nil || u.Scheme == "" || u.Host == "" {
			log.Warnf("ignoring invalid allowed origin %q", origin)
			continue
		}

		normalized := strings.ToLower(u.Scheme + "://" + u.Host)
		allowed[normalized] = struct{}{}
	}

	return func(r *http.Request) bool {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			// Browsers should send Origin for websocket requests.
			// Rejecting empty Origin is stricter and usually safer for browser-only websocket APIs.
			return false
		}

		u, err := url.Parse(origin)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return false
		}

		normalized := strings.ToLower(u.Scheme + "://" + u.Host)

		_, ok := allowed[normalized]
		if !ok {
			log.Warnf("rejecting websocket request from origin %q", origin)
		}

		return ok
	}
}
