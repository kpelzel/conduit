// Copyright 2026. Triad National Security, LLC. All rights reserved.

package etcd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	proto "github.com/lanl/conduit/api"
	"github.com/lanl/conduit/defaults"
	"github.com/spf13/viper"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	listenTimeout = 5 * time.Second
)

// StartWatchChannels will start go routines to watch the transfer and lease areas in etcd
func (em *ETCDManager) StartWatchChannels(rev int64, cancel context.CancelCauseFunc) {
	// setup long running watch channels
	tc, tWatchCancel := em.GetWatchChannelPrefix(proto.TransferPrefix, rev)
	lc, lWatchCancel := em.GetWatchChannelPrefix(proto.LeasePrefix, rev)
	ec, eWatchCancel := em.GetWatchChannelPrefix(proto.ErrorsPrefix, rev)

	tctx, tCancel := context.WithCancelCause(context.Background())
	lctx, lCancel := context.WithCancelCause(context.Background())
	ectx, eCancel := context.WithCancelCause(context.Background())
	go em.watchTransfers(tc, tCancel)
	go em.watchLeases(lc, lCancel)
	go em.watchErrant(ec, eCancel)

	var err error

	select {
	case <-tctx.Done():
		err = fmt.Errorf("error detected while watching transfers: %v", context.Cause(tctx))
	case <-lctx.Done():
		err = fmt.Errorf("error detected while watching leases: %v", context.Cause(lctx))
	case <-ectx.Done():
		err = fmt.Errorf("error detected while watching errants: %v", context.Cause(ectx))
	}

	em.log.Error(err)

	tWatchCancel()
	lWatchCancel()
	eWatchCancel()

	cancel(err)
}

// watchTransfers is the go routine that watches changes to the transfers area in etcd. It sends these events to any subscribers
func (em *ETCDManager) watchTransfers(wc <-chan clientv3.WatchResponse, cancel context.CancelCauseFunc) {
	// wc, cancel := em.GetWatchChannelPrefix(proto.TransferPrefix)
	for wresp := range wc {
		em.tmutex.RLock()
		for id, tclient := range em.tclients {
			select {
			case tclient <- wresp:
			case <-time.After(listenTimeout):
				// if we run into this error, it needs to be fixed on the listener end. Make sure all subscribers unsubscribe when they are finished listening
				em.log.Errorf("timeout sending transfer updates to listener[%s]", id)
				// go em.UnsubscribeFromTransfers(id)
			}
		}
		em.tmutex.RUnlock()
		if wresp.Canceled {
			cancel(fmt.Errorf("transfer watch channel cancelled: %v", wresp.Err()))
			em.log.Error(wresp.Err())
			return
		}
	}

	terr := fmt.Errorf("transfer watch channel closed unexpectedly")
	em.log.Error(terr)
	cancel(terr)
}

// SubscribeToTransfers will return a channel for any service to subscribe to transfers events
func (em *ETCDManager) SubscribeToTransfers(id uuid.UUID) <-chan clientv3.WatchResponse {
	c := make(chan clientv3.WatchResponse, 1000)
	em.tmutex.Lock()
	em.tclients[id] = c
	em.tmutex.Unlock()
	return c
}

// UnsubscribeFromTransfers will remove a service from the list of transfers subscribers
func (em *ETCDManager) UnsubscribeFromTransfers(id uuid.UUID) {
	em.tmutex.Lock()
	delete(em.tclients, id)
	em.tmutex.Unlock()
}

// watchLeases is the go routine that watches changes to the leases area in etcd. It sends these events to any subscribers
func (em *ETCDManager) watchLeases(wc <-chan clientv3.WatchResponse, cancel context.CancelCauseFunc) {
	// wc, cancel := em.GetWatchChannelPrefix(proto.LeasePrefix)
	for wresp := range wc {
		em.lmutex.RLock()
		for id, lclient := range em.lclients {
			select {
			case lclient <- wresp:
			case <-time.After(listenTimeout):
				// if we run into this error, it needs to be fixed on the listener end. Make sure all subscribers unsubscribe when they are finished listening
				em.log.Errorf("timeout sending lease updates to listener[%s]", id)
				// go em.UnsubscribeFromTransfers(id)
			}
		}
		em.lmutex.RUnlock()
		if wresp.Canceled {
			cancel(fmt.Errorf("lease watch channel cancelled"))
			em.log.Error(wresp.Err())
			return
		}
	}

	lerr := fmt.Errorf("lease watch channel closed unexpectedly")
	em.log.Error(lerr)
	cancel(lerr)
}

// SubscribeToLeases will return a channel for any service to subscribe to leases events
func (em *ETCDManager) SubscribeToLeases(id uuid.UUID) <-chan clientv3.WatchResponse {
	c := make(chan clientv3.WatchResponse, 1000)
	em.lmutex.Lock()
	em.lclients[id] = c
	em.lmutex.Unlock()
	return c
}

// UnsubscribeFromTransfers will remove a service from the list of leases subscribers
func (em *ETCDManager) UnsubscribeFromLeases(id uuid.UUID) {
	em.lmutex.Lock()
	delete(em.lclients, id)
	em.lmutex.Unlock()
}

// WaitTransfersActive will block until all transfers with "ids" are no longer active
func (em *ETCDManager) WaitTransfersActive(ids []uuid.UUID, ctx context.Context) error {
	// create watch uuid
	watchID, err := uuid.NewUUID()
	if err != nil {
		return fmt.Errorf("failed to create uuid: %v", err)
	}

	// make a map for active transfers. transfers will get deleted as they become inactive
	activeTransfers := make(map[uuid.UUID]bool)
	atLock := sync.RWMutex{}
	// previousUpdates is used to keep track of any updates that happened right now before we start watching
	previousUpdates := make(map[uuid.UUID]bool)

	// add all watched transfers to both activeTransfers and previousUpdates
	for _, id := range ids {
		activeTransfers[id] = true
		previousUpdates[id] = true
	}

	// we know we started watching on the channel when we get a message on this channel
	goStartedWatching := make(chan bool, 1)
	// we know the watch is complete when we get a message on this done channel
	goDone := make(chan error, 1)

	go func() {
		// subscribe to transfer updates
		wch := em.SubscribeToTransfers(watchID)
		defer em.UnsubscribeFromTransfers(watchID)
		goStartedWatching <- true

		// watch for transfer events in etcd
		for {
			select {
			case wresp, ok := <-wch:
				if !ok {
					goDone <- fmt.Errorf("transfer watch channel closed unexpectedly")
					return
				}
				for _, ev := range wresp.Events {
					// if the event is an active key check if its one of our watched transfers
					if ev.Type == mvccpb.PUT && strings.HasSuffix(string(ev.Kv.Key), proto.ActiveKey) {
						id, _, err := proto.ParseETCDTransfersKey(string(ev.Kv.Key))
						if err != nil {
							em.log.Errorf("failed to parse transfer id from etcd key: %+v", err)
							continue
						}
						found := false
						for _, i := range ids {
							if i == id {
								found = true
								break
							}
						}
						// it is one of our watched transfers. if it's no longer active, remove from activeTransfers
						if found {
							active, err := strconv.ParseBool(string(ev.Kv.Value))
							if err != nil {
								em.log.Errorf("failed to parse active bool: %+v", err)
								continue
							}

							if !active {
								atLock.Lock()
								delete(activeTransfers, id)
								atLock.Unlock()
							}
						}
					}
				}
				// check if there are no more active transfers and break if there aren't
				atLock.RLock()
				if len(activeTransfers) == 0 {
					goDone <- nil
					atLock.RUnlock()
					return
				}
				atLock.RUnlock()
			case <-ctx.Done():
				goDone <- context.Canceled
				return
			}
		}
	}()

	// wait till we actually start watching transfers
	<-goStartedWatching

	// check every watched transfer to see if it already completed before we started watching
	for _, id := range ids {
		it := proto.IncompleteTransfer(&proto.TransferDetails{TransferID: id.String()})
		a, err := em.GetActive(it)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				a = false
			} else {
				em.log.Errorf("failed to get transfer active state from etcd: %v", err)
				continue
			}
		}
		previousUpdates[id] = a
	}

	// remove any inactive transfers that we found from our activeTransfers map
	atLock.Lock()
	for id, a := range previousUpdates {
		if !a {
			delete(activeTransfers, id)
		}
	}
	if len(activeTransfers) == 0 {
		atLock.Unlock()
		return nil
	}
	atLock.Unlock()

	// wait for the watch to complete
	err = <-goDone
	if err != nil {
		if err.Error() == context.Canceled.Error() {
			return err
		}
		return fmt.Errorf("error while watching transfers: %v", err)
	}

	return nil
}

// UpdateExpiryConstantly will update a transfers expiry every 10 seconds. The new expiry will be the configured ExpiryAdvance duration from the current time
func (em *ETCDManager) UpdateExpiryConstantly(it proto.IncompleteTransfer, stopChan <-chan bool) {
	em.log.Debugf("constantly updating expiry for transfer[%v] every %v seconds", it.GetTransferID(), 10)

	_, err, _ := em.UpdateExpiryOnce(it)
	if err != nil {
		em.log.Error(err)
	}

	for {
		select {
		case <-stopChan:
			em.log.Debugf("finished constantly updating expiry for transfer[%v]", it.GetTransferID())
			return
		case <-time.After(10 * time.Second):
			_, err, _ := em.UpdateExpiryOnce(it)
			if err != nil {
				em.log.Error(err)
			}
		}
	}
}

// UpdateExpiryOnce will update a transfers expiry one time. The new expiry will be the configured ExpiryAdvance duration from the current time
func (em *ETCDManager) UpdateExpiryOnce(it proto.IncompleteTransfer) (succeeded bool, err error, newExpiry *timestamppb.Timestamp) {
	newExpiry = timestamppb.New(time.Now().Add(viper.GetDuration(defaults.ConfigExpiryAdvanceKey)))

	// check if the transfer has an error before updating the expiry key
	resp, err := em.RetryTxn(
		&[]clientv3.Cmp{
			clientv3.Compare(clientv3.Value(it.ETCDErrorKey()), "=", proto.Error_ERROR_NONE.String()),
			clientv3.Compare(clientv3.Value(it.ETCDActiveKey()), "=", strconv.FormatBool(true)),
		},
		&[]clientv3.Op{
			clientv3.OpPut(it.ETCDExpiryKey(), newExpiry.AsTime().Format(time.RFC3339)),
		},
		defaults.MaxRetries,
		defaults.RetryDelay,
	)
	if err != nil {
		return false, fmt.Errorf("failed to update expiry in etcd for transfer[%s]: %s ", it.GetTransferID(), err), newExpiry
	}

	return resp.Succeeded, nil, newExpiry
}

// watchErrant is the go routine that watches changes to the errors area in etcd. It sends these events to any subscribers
func (em *ETCDManager) watchErrant(wc <-chan clientv3.WatchResponse, cancel context.CancelCauseFunc) {

	// because errors can be compacted over, we might not get existing errors at the start of conduit
	// do a prefix pull first and send it to the client before sending the watch events
	resp, err := em.GetPrefix(proto.ErrorsPrefix)
	if err != nil {
		em.log.Errorf("failed to get errants before watching for changes: %v", err)
	}
	events := []*clientv3.Event{}
	for _, kv := range resp.Kvs {
		events = append(events, &clientv3.Event{
			Kv: kv,
		})
	}
	for id, eclient := range em.eclients {
		select {
		case eclient <- clientv3.WatchResponse{Events: events}:
		case <-time.After(listenTimeout):
			// if we run into this error, it needs to be fixed on the listener end. Make sure all subscribers unsubscribe when they are finished listening
			em.log.Errorf("timeout sending error updates to listener[%s]", id)
		}
	}

	for wresp := range wc {
		em.emutex.RLock()
		for id, eclient := range em.eclients {
			select {
			case eclient <- wresp:
			case <-time.After(listenTimeout):
				// if we run into this error, it needs to be fixed on the listener end. Make sure all subscribers unsubscribe when they are finished listening
				em.log.Errorf("timeout sending error updates to listener[%s]", id)
			}
		}
		em.emutex.RUnlock()
		if wresp.Canceled {
			cancel(fmt.Errorf("errant watch channel cancelled"))
			em.log.Error(wresp.Err())
			return
		}
	}

	eerr := fmt.Errorf("errant watch channel closed unexpectedly")
	em.log.Error(eerr)
	cancel(eerr)
}

// SubscribeToErrant will return a channel for any service to subscribe to error events
func (em *ETCDManager) SubscribeToErrant(id uuid.UUID) <-chan clientv3.WatchResponse {
	c := make(chan clientv3.WatchResponse, 1000)
	em.emutex.Lock()
	em.eclients[id] = c
	em.emutex.Unlock()
	return c
}

// UnsubscribeFromErrant will remove a service from the list of error subscribers
func (em *ETCDManager) UnsubscribeFromErrant(id uuid.UUID) {
	em.emutex.Lock()
	delete(em.eclients, id)
	em.emutex.Unlock()
}
