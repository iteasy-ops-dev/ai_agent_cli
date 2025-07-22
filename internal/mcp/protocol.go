package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
	"github.com/iteasy-ops-dev/syseng-agent/pkg/utils"
)

type MCPProcess struct {
	server    *types.MCPServer
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	tools     map[string]Tool
	responses map[int]chan *MCPResponse
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	nextID    int
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"inputSchema"`
}

type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeParams struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    map[string]bool   `json:"capabilities"`
	ClientInfo      map[string]string `json:"clientInfo"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

func NewMCPProcess(server *types.MCPServer) *MCPProcess {
	ctx, cancel := context.WithCancel(context.Background())
	return &MCPProcess{
		server:    server,
		tools:     make(map[string]Tool),
		responses: make(map[int]chan *MCPResponse),
		ctx:       ctx,
		cancel:    cancel,
		nextID:    1,
	}
}

func (p *MCPProcess) Start() error {
	if p.server.Transport != "stdio" {
		return fmt.Errorf("only stdio transport is supported")
	}

	// Start the MCP server process using the server URL
	args := strings.Fields(p.server.URL)
	if len(args) == 0 {
		return fmt.Errorf("invalid server URL: %s", p.server.URL)
	}
	p.cmd = exec.CommandContext(p.ctx, args[0], args[1:]...)

	var err error
	p.stdin, err = p.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	p.stdout, err = p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	p.stderr, err = p.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Start output readers
	go p.readOutput()
	go p.readErrors()

	// Initialize the MCP connection
	if err := p.initialize(); err != nil {
		return fmt.Errorf("failed to initialize MCP connection: %w", err)
	}

	// Discover available tools
	if err := p.discoverTools(); err != nil {
		return fmt.Errorf("failed to discover tools: %w", err)
	}

	return nil
}

func (p *MCPProcess) Stop() error {
	p.cancel()

	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}

	return nil
}

func (p *MCPProcess) initialize() error {
	initParams := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities: map[string]bool{
			"tools": true,
		},
		ClientInfo: map[string]string{
			"name":    "syseng-agent",
			"version": "1.0.0",
		},
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  initParams,
	}

	resp, err := p.sendRequest(req)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialization error: %s", resp.Error.Message)
	}

	// Send initialized notification
	notification := MCPRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	return p.sendNotification(notification)
}

func (p *MCPProcess) discoverTools() error {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}

	resp, err := p.sendRequest(req)
	if err != nil {
		return fmt.Errorf("tools discovery failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("tools discovery error: %s", resp.Error.Message)
	}

	// Parse tools from response
	if result, ok := resp.Result.(map[string]interface{}); ok {
		if tools, ok := result["tools"].([]interface{}); ok {
			p.mu.Lock()
			for _, toolData := range tools {
				if toolMap, ok := toolData.(map[string]interface{}); ok {
					tool := Tool{
						Name:        utils.GetString(toolMap, "name"),
						Description: utils.GetString(toolMap, "description"),
						Schema:      utils.GetMap(toolMap, "inputSchema"),
					}
					p.tools[tool.Name] = tool
				}
			}
			p.mu.Unlock()
		}
	}

	return nil
}

func (p *MCPProcess) CallTool(name string, arguments map[string]interface{}) (interface{}, error) {
	p.mu.RLock()
	_, exists := p.tools[name]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	params := ToolCallParams{
		Name:      name,
		Arguments: arguments,
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      int(time.Now().Unix()),
		Method:  "tools/call",
		Params:  params,
	}

	resp, err := p.sendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tool call error: %s", resp.Error.Message)
	}

	return resp.Result, nil
}

func (p *MCPProcess) GetTools() []Tool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tools := make([]Tool, 0, len(p.tools))
	for _, tool := range p.tools {
		tools = append(tools, tool)
	}

	return tools
}

func (p *MCPProcess) sendRequest(req MCPRequest) (*MCPResponse, error) {
	// Assign ID if not set
	if req.ID == 0 {
		p.mu.Lock()
		req.ID = p.nextID
		p.nextID++
		p.mu.Unlock()
	}

	// Create response channel
	respCh := make(chan *MCPResponse, 1)
	p.mu.Lock()
	p.responses[req.ID] = respCh
	p.mu.Unlock()

	// Marshal and send request
	data, err := json.Marshal(req)
	if err != nil {
		p.mu.Lock()
		delete(p.responses, req.ID)
		p.mu.Unlock()
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	data = append(data, '\n')

	if _, err := p.stdin.Write(data); err != nil {
		p.mu.Lock()
		delete(p.responses, req.ID)
		p.mu.Unlock()
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Wait for response with timeout
	select {
	case resp := <-respCh:
		return resp, nil
	case <-time.After(10 * time.Second):
		p.mu.Lock()
		delete(p.responses, req.ID)
		p.mu.Unlock()
		return nil, fmt.Errorf("request timeout")
	case <-p.ctx.Done():
		return nil, fmt.Errorf("process cancelled")
	}
}

func (p *MCPProcess) sendNotification(req MCPRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	data = append(data, '\n')

	if _, err := p.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write notification: %w", err)
	}

	return nil
}

func (p *MCPProcess) readOutput() {
	scanner := bufio.NewScanner(p.stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse JSON-RPC responses
		var resp MCPResponse
		if err := json.Unmarshal([]byte(line), &resp); err == nil {
			// Handle response with ID (request response)
			if resp.ID != 0 {
				p.mu.Lock()
				if ch, exists := p.responses[resp.ID]; exists {
					select {
					case ch <- &resp:
					default:
						// Channel full, skip
					}
					delete(p.responses, resp.ID)
				}
				p.mu.Unlock()
			}
			// Ignore notifications (no ID)
		}
	}
}

func (p *MCPProcess) readErrors() {
	scanner := bufio.NewScanner(p.stderr)
	for scanner.Scan() {
		line := scanner.Text()

		// Filter out normal loading messages from desktop-commander
		if strings.Contains(line, "Loading server.ts") ||
			strings.Contains(line, "Setting up request handlers") ||
			strings.Contains(line, "initialized") ||
			strings.Contains(line, "Loading configuration") ||
			strings.Contains(line, "Configuration loaded") ||
			strings.Contains(line, "Connecting server") ||
			strings.Contains(line, "Server connected") ||
			strings.Contains(line, "Generating tools list") {
			continue // Skip normal loading messages
		}

		// Only show actual errors
		fmt.Printf("MCP Error: %s\n", line)
	}
}

