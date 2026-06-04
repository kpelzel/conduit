package mcp

import (
	"context"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	proto "github.com/lanl/conduit/api"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetTimeParams defines the parameters for the cityTime tool.
type StartTransferParams struct {
	Action      string   `json:"action" jsonschema:"action for the transfer"`
	Source      []string `json:"source" jsonschema:"list of paths of source files/directories"`
	Destination string   `json:"destination" jsonschema:"path of destination file/directory"`
}

type GetTransferStatusParams struct {
	TransferID string `json:"transfer_id" jsonschema:"Transfer ID required to get status"`
}

type StartTransferResult struct {
	TransferID string `json:"transfer_id" jsonschema:"ID of the transfer"`
	Successful bool   `json:"state" jsonschema:"signifies wether the transfer was successfully submitted or not. It does not indicate that the transfer successfully completed."`
}

type GetTransferStatusResult struct {
	TransferID string `json:"transfer_id" jsonschema:"ID of the transfer"`
	State      string `json:"state" jsonschema:"current state of the transfer"`
	Active     bool   `json:"active" jsonschema:"whether the transfer is still active or not"`
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
		"COPY",
		"MOVE",
		"RECURSIVE_COPY",
		"RECURSIVE_MOVE",
	}

	mcpsdk.AddTool(m.mcpServer, &mcpsdk.Tool{
		Name:         "start_transfer",
		Description:  "Start a new data transfer from a SOURCE to a DESTINATION. Returns transfer_id. After a successful call, immediately call get_transfer_status with the returned transfer_id to verify the initial state.",
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

	action, ok := proto.Action_value[params.Action]
	if !ok {
		return nil, nil, fmt.Errorf("provided an invalid action: %v", params.Action)
	}

	tr := &proto.TransferRequest{
		User:        info.UserID,
		Action:      proto.Action(action),
		Source:      params.Source,
		Destination: params.Destination,
	}

	m.log.Debugf("mcp start transfer request: %+v", tr)

	resp, err := m.conduitClient.StartTransfer(ctx, tr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start transfer: %w", err)
	}

	return nil, &StartTransferResult{
		TransferID: resp.GetTransferID(),
		Successful: true,
	}, nil
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

	return nil, &GetTransferStatusResult{
		TransferID: params.TransferID,
		State:      details.GetState().String(),
		Active:     details.GetActive(),
	}, nil
}
