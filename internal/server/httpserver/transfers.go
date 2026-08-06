// Copyright 2026. Triad National Security, LLC. All rights reserved.

package httpserver

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lanl/conduit/api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// GET "/transfers/{transferID}"
func (h *HTTPServer) getTransferByID(wr http.ResponseWriter, req *http.Request, username string) {
	transferID := req.PathValue("transferID")
	if transferID == "" {
		http.Error(wr, "missing transfer ID", http.StatusBadRequest)
		return
	}

	h.queryTransfersByIDList(wr, req, username, []string{transferID})
}

// GET "/transfers"
func (h *HTTPServer) getTransfers(wr http.ResponseWriter, req *http.Request, username string) {
	h.queryTransfersByIDList(wr, req, username, nil)
}

func (h *HTTPServer) queryTransfersByIDList(wr http.ResponseWriter, req *http.Request, username string, transferIDs []string) {
	qo := &api.QueryOptions{
		User:           username,
		QueryOperation: api.QueryOperation_QUERY_OR,
	}

	// add transfer ids if any were provided
	if len(transferIDs) > 0 {
		qo.QueryMap = map[string]string{
			"TransferID": strings.Join(transferIDs, "|"),
		}
	}

	mtd, err := h.conduitClient.Query(req.Context(), qo)
	if err != nil {
		http.Error(wr, fmt.Sprintf("failed to query conduit for transfers: %v", err), http.StatusInternalServerError)
		return
	}

	writeProto(wr, req, mtd)
}

// POST "/transfers/query"
func (h *HTTPServer) queryTransfers(wr http.ResponseWriter, req *http.Request, username string) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(wr, fmt.Sprintf("failed to read request body: %v", err), http.StatusInternalServerError)
		return
	}

	qo := &api.QueryOptions{}

	err = proto.Unmarshal(body, qo)
	if err != nil {
		http.Error(wr, fmt.Sprintf("failed to unmarshal request body: %v", err), http.StatusBadRequest)
		return
	}

	// set user to the one that we authed with
	qo.User = username

	mtd, err := h.conduitClient.Query(req.Context(), qo)
	if err != nil {
		http.Error(wr, fmt.Sprintf("failed to query conduit for transfers: %v", err), http.StatusInternalServerError)
		return
	}

	writeProto(wr, req, mtd)
}

// POST "/transfers/{transferID}/abort"
func (h *HTTPServer) abortTransfer(wr http.ResponseWriter, req *http.Request, username string) {
	transferID := req.PathValue("transferID")
	if transferID == "" {
		http.Error(wr, "missing transfer ID", http.StatusBadRequest)
		return
	}

	h.abortTransferIDList(wr, req, username, []string{transferID})
}

// POST "/transfers/abort"
func (h *HTTPServer) abortTransfers(wr http.ResponseWriter, req *http.Request, username string) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(wr, fmt.Sprintf("failed to read request body: %v", err), http.StatusInternalServerError)
		return
	}

	tIds := &api.TransferIds{}

	err = proto.Unmarshal(body, tIds)
	if err != nil {
		http.Error(wr, fmt.Sprintf("failed to unmarshal request body: %v", err), http.StatusBadRequest)
		return
	}

	h.abortTransferIDList(wr, req, username, tIds.GetValue())
}

func (h *HTTPServer) abortTransferIDList(wr http.ResponseWriter, req *http.Request, username string, transferIDs []string) {
	if len(transferIDs) == 0 {
		http.Error(wr, "missing transfer IDs", http.StatusBadRequest)
		return
	}

	mtd, err := h.conduitClient.StopTransfer(req.Context(), &api.TransferIds{
		User:  username,
		Value: transferIDs,
	})
	if err != nil {
		http.Error(wr, fmt.Sprintf("failed to abort transfers[%v]: %v", transferIDs, err), http.StatusInternalServerError)
		return
	}

	writeProto(wr, req, mtd)
}

// POST "/transfers"
func (h *HTTPServer) startTransfer(wr http.ResponseWriter, req *http.Request, username string) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(wr, fmt.Sprintf("failed to read request body: %v", err), http.StatusInternalServerError)
		return
	}

	tr := &api.TransferRequest{}

	err = proto.Unmarshal(body, tr)
	if err != nil {
		http.Error(wr, fmt.Sprintf("failed to unmarshal request body: %v", err), http.StatusBadRequest)
		return
	}

	tr.User = username

	mtd, err := h.conduitClient.StartTransfer(req.Context(), tr)
	if err != nil {
		http.Error(wr, fmt.Sprintf("failed to start transfer: %v", err), http.StatusInternalServerError)
		return
	}

	writeProto(wr, req, mtd)
}

// writeResponse writes the response in the format requested by the client via Accept header
// Supports application/json (default) and application/x-protobuf
func writeProto(wr http.ResponseWriter, req *http.Request, msg proto.Message) {
	accept := req.Header.Get("Accept")

	// Check if client explicitly wants Protobuf
	if strings.Contains(accept, "application/x-protobuf") || strings.Contains(accept, "application/protobuf") {
		writeProtobuf(wr, msg)
		return
	}

	// Default to JSON (more user-friendly for HTTP APIs)
	writeJSON(wr, msg)
}

func writeJSON(wr http.ResponseWriter, msg proto.Message) {
	// Use protojson for proper proto3 JSON encoding
	marshaler := protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}

	b, err := marshaler.Marshal(msg)
	if err != nil {
		http.Error(wr, fmt.Sprintf("failed to marshal JSON response: %v", err), http.StatusInternalServerError)
		return
	}

	wr.Header().Set("Content-Type", "application/json")
	wr.WriteHeader(http.StatusOK)
	_, _ = wr.Write(b)
}

func writeProtobuf(wr http.ResponseWriter, msg proto.Message) {
	b, err := proto.Marshal(msg)
	if err != nil {
		http.Error(wr, fmt.Sprintf("failed to marshal protobuf response: %v", err), http.StatusInternalServerError)
		return
	}

	wr.Header().Set("Content-Type", "application/x-protobuf")
	wr.WriteHeader(http.StatusOK)
	_, _ = wr.Write(b)
}
