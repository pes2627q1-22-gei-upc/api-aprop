package mcp_client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const clientName = "api-aprop"

// MCPClient connects to the catalogue MCP server over Streamable HTTP with a bearer token.
type MCPClient struct {
	client     *mcp.Client
	endpoint   string
	httpClient *http.Client
}

var _ Client = (*MCPClient)(nil)

func NewMCPClient(httpClient *http.Client, endpoint string, token string) *MCPClient {
	base := httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	authenticated := *httpClient
	authenticated.Transport = bearerTransport{base: base, token: token}
	return &MCPClient{
		client:     mcp.NewClient(&mcp.Implementation{Name: clientName}, nil),
		endpoint:   endpoint,
		httpClient: &authenticated,
	}
}

func (c *MCPClient) Connect(ctx context.Context) (Session, error) {
	session, err := c.client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             c.endpoint,
		HTTPClient:           c.httpClient,
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to mcp server: %w", err)
	}
	return &mcpSession{session: session}, nil
}

type mcpSession struct {
	session *mcp.ClientSession
}

func (s *mcpSession) ListTools(ctx context.Context) ([]Tool, error) {
	res, err := s.session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("list mcp tools: %w", err)
	}
	tools := make([]Tool, 0, len(res.Tools))
	for _, t := range res.Tools {
		tools = append(tools, Tool{Name: t.Name, Description: t.Description, InputSchema: t.InputSchema})
	}
	return tools, nil
}

func (s *mcpSession) CallTool(ctx context.Context, name string, arguments json.RawMessage) (ToolResult, error) {
	if len(arguments) == 0 || string(arguments) == "null" {
		arguments = json.RawMessage(`{}`)
	}
	res, err := s.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		return ToolResult{}, fmt.Errorf("call mcp tool %s: %w", name, err)
	}
	result := ToolResult{IsError: res.IsError}
	var text strings.Builder
	for _, content := range res.Content {
		if t, ok := content.(*mcp.TextContent); ok {
			text.WriteString(t.Text)
		}
	}
	result.Text = text.String()
	if res.StructuredContent != nil {
		if result.Structured, err = json.Marshal(res.StructuredContent); err != nil {
			return ToolResult{}, fmt.Errorf("encode mcp tool %s result: %w", name, err)
		}
	}
	return result, nil
}

func (s *mcpSession) Close() error {
	return s.session.Close()
}

type bearerTransport struct {
	base  http.RoundTripper
	token string
}

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(r)
}
