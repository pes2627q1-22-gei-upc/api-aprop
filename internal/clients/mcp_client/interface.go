package mcp_client

import (
	"context"
	"encoding/json"
)

// Tool is a tool offered by the MCP server. InputSchema is a JSON schema object.
type Tool struct {
	Name        string
	Description string
	InputSchema any
}

// ToolResult is the result of a tool call. IsError results are meant to be shown to the model.
type ToolResult struct {
	Text       string
	Structured json.RawMessage
	IsError    bool
}

// Session is an open connection to the MCP server.
type Session interface {
	ListTools(ctx context.Context) ([]Tool, error)
	CallTool(ctx context.Context, name string, arguments json.RawMessage) (ToolResult, error)
	Close() error
}

// Client opens sessions with the MCP server.
type Client interface {
	Connect(ctx context.Context) (Session, error)
}
