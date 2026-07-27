package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	proto "github.com/lanl/conduit/api"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/encoding/protojson"
)

// GetTimeParams defines the parameters for the cityTime tool.
type StartTransferParams struct {
	Action      string   `json:"action" jsonschema:"transfer action. Use COPY by default unless the user explicitly asks to move or recursively copy/move. Valid values: CONDUIT_COPY, CONDUIT_MOVE"`
	Source      []string `json:"source" jsonschema:"one or more source file or directory paths"`
	Destination string   `json:"destination" jsonschema:"destination file or directory path"`
}

type GetTransferStatusParams struct {
	TransferID string `json:"transfer_id" jsonschema:"Transfer ID required to get status"`
}

type StartTransferResult struct {
	Submitted  bool           `json:"submitted" jsonschema:"true if the transfer request was successfully submitted to Conduit"`
	Message    string         `json:"message" jsonschema:"human-readable summary of the transfer submission"`
	TransferID string         `json:"transfer_id" jsonschema:"ID of the transfer"`
	Details    map[string]any `json:"details" jsonschema:"full Conduit TransferDetails object serialized with protojson"`
}

type GetTransferStatusResult struct {
	Message    string         `json:"message" jsonschema:"human-readable transfer status summary"`
	TransferID string         `json:"transfer_id" jsonschema:"ID of the transfer"`
	State      string         `json:"state" jsonschema:"current transfer state"`
	Active     bool           `json:"active" jsonschema:"whether the transfer is still active"`
	Details    map[string]any `json:"details" jsonschema:"full Conduit TransferDetails object serialized with protojson"`
}

func (m *MCPServer) registerTools() error {
	m.mcpServer.AddReceivingMiddleware(m.createLoggingMiddleware())

	startParamsSchema, err := jsonschema.For[StartTransferParams](nil)
	if err != nil {
		return fmt.Errorf("create start_transfer params schema: %w", err)
	}

	startResultSchema, err := jsonschema.For[StartTransferResult](nil)
	if err != nil {
		return fmt.Errorf("create start_transfer result schema: %w", err)
	}

	startParamsSchema.Properties["action"].Enum = []any{
		"CONDUIT_COPY",
		"CONDUIT_MOVE",
	}

	mcpsdk.AddTool(m.mcpServer, &mcpsdk.Tool{
		Name:         "start_transfer",
		Description:  "Start a Conduit file transfer. Use this when the user asks to copy, move, or transfer files or directories. On success, this tool has already submitted the transfer to Conduit and returns a transfer_id. The assistant should tell the user the transfer was submitted successfully and include the transfer_id. If the user did not specify an action, use COPY.",
		InputSchema:  startParamsSchema,
		OutputSchema: startResultSchema,
	}, m.startTransfer)

	statusResultSchema, err := jsonschema.For[GetTransferStatusResult](nil)
	if err != nil {
		return fmt.Errorf("create get_transfer_status result schema: %w", err)
	}

	statusResultSchema.Properties["state"].Enum = []any{
		"TRANSFER_NONE",
		"TRANSFER_ERROR",
		"TRANSFER_ABORT",
		"TRANSFER_ABORTED",
		"TRANSFER_INIT",
		"TRANSFER_INIT_COMPLETE",
		"TRANSFER_VALIDATION_READY",
		"TRANSFER_VALIDATION_SUBMITTED",
		"TRANSFER_VALIDATING",
		"TRANSFER_VALIDATION_COMPLETE",
		"TRANSFER_WAITING_FOR_LEASE",
		"TRANSFER_LEASE_ACQUIRED",
		"TRANSFER_SETUP_READY",
		"TRANSFER_SETUP_SUBMITTED",
		"TRANSFER_SETUP",
		"TRANSFER_SETUP_COMPLETE",
		"TRANSFER_DATA_READY",
		"TRANSFER_DATA_SUBMITTED",
		"TRANSFER_DATA_TRANSFERRING",
		"TRANSFER_DATA_COMPLETE",
		"TRANSFER_TEARDOWN_READY",
		"TRANSFER_TEARDOWN_SUBMITTED",
		"TRANSFER_TEARDOWN",
		"TRANSFER_TEARDOWN_COMPLETE",
		"TRANSFER_FINALIZED",
	}

	statusParamsSchema, err := jsonschema.For[GetTransferStatusParams](nil)
	if err != nil {
		return fmt.Errorf("create get_transfer_status params schema: %w", err)
	}

	mcpsdk.AddTool(m.mcpServer, &mcpsdk.Tool{
		Name:         "get_transfer_status",
		Description:  "Get the status of an existing transfer by transfer_id. Use immediately after start_transfer and whenever the user asks for an updated transfer state.",
		InputSchema:  statusParamsSchema,
		OutputSchema: statusResultSchema,
	}, m.getTransferStatus)

	return nil
}

func (m *MCPServer) startTransfer(ctx context.Context, req *mcpsdk.CallToolRequest, params *StartTransferParams) (*mcpsdk.CallToolResult, *StartTransferResult, error) {
	info := mcpauth.TokenInfoFromContext(ctx)
	if info == nil || info.UserID == "" {
		return nil, nil, fmt.Errorf("no authenticated user provided")
	}

	tr := &proto.TransferRequest{
		User:        info.UserID,
		Action:      params.Action,
		Source:      params.Source,
		Destination: params.Destination,
	}

	m.log.Debugf("mcp start transfer request: %+v", tr)

	resp, err := m.conduitClient.StartTransfer(ctx, tr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start transfer: %w", err)
	}

	details, err := transferDetailsMap(resp)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal transfer details: %w", err)
	}

	transferID := resp.GetTransferID()

	message := fmt.Sprintf(
		"Transfer submitted successfully. transfer_id=%s action=%s source=%v destination=%s state=%s active=%t.",
		transferID,
		resp.GetAction(),
		resp.GetSource(),
		resp.GetDestination(),
		resp.GetState().String(),
		resp.GetActive(),
	)

	out := &StartTransferResult{
		Submitted:  true,
		Message:    message,
		TransferID: transferID,
		Details:    details,
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{Text: message},
		},
	}, out, nil
}

func (m *MCPServer) getTransferStatus(ctx context.Context, req *mcpsdk.CallToolRequest, params *GetTransferStatusParams) (*mcpsdk.CallToolResult, *GetTransferStatusResult, error) {
	info := mcpauth.TokenInfoFromContext(ctx)
	if info == nil || info.UserID == "" {
		return nil, nil, fmt.Errorf("no authenticated user provided")
	}

	qo := &proto.QueryOptions{
		User:           info.UserID,
		QueryOperation: proto.QueryOperation_QUERY_OR,
		QueryMap:       map[string]string{"TransferID": params.TransferID},
	}

	resp, err := m.conduitClient.Query(ctx, qo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query for transfer: %w", err)
	}

	details := resp.GetDetails()[params.TransferID]
	if details == nil {
		return nil, nil, fmt.Errorf("transfer not found: %s", params.TransferID)
	}

	detailsJSON, err := transferDetailsMap(details)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal transfer details: %w", err)
	}

	message := fmt.Sprintf(
		"Transfer %s status: state=%s active=%t error=%s error_message=%q.",
		params.TransferID,
		details.GetState().String(),
		details.GetActive(),
		details.GetError().String(),
		details.GetErrorMessage(),
	)

	return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: message},
			},
		}, &GetTransferStatusResult{
			Message:    message,
			TransferID: params.TransferID,
			State:      details.GetState().String(),
			Active:     details.GetActive(),
			Details:    detailsJSON,
		}, nil
}

// createLoggingMiddleware creates an MCP middleware that logs method calls.
func (m *MCPServer) createLoggingMiddleware() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(
			ctx context.Context,
			method string,
			req mcp.Request,
		) (mcp.Result, error) {
			start := time.Now()
			sessionID := req.GetSession().ID()

			// Log request details.
			m.log.Debugf("[REQUEST] Session: %s | Method: %s",
				sessionID,
				method)

			// Call the actual handler.
			result, err := next(ctx, method, req)

			// Log response details.
			duration := time.Since(start)

			if err != nil {
				m.log.Errorf("[RESPONSE] Session: %s | Method: %s | Status: ERROR | Duration: %v | Error: %v",
					sessionID,
					method,
					duration,
					err)
			} else {
				m.log.Debugf("[RESPONSE] Session: %s | Method: %s | Status: OK | Duration: %v",
					sessionID,
					method,
					duration)
			}

			return result, err
		}
	}
}

func transferDetailsMap(td *proto.TransferDetails) (map[string]any, error) {
	b, err := protojson.MarshalOptions{
		UseEnumNumbers:  false,
		EmitUnpopulated: true,
		UseProtoNames:   false,
	}.Marshal(td)
	if err != nil {
		return nil, err
	}

	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}

	return out, nil
}
