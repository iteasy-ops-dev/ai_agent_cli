package mcp

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/iteasy-ops-dev/syseng-agent/internal/storage"
	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
)

// debugPrint prints debug messages only if DEBUG environment variable is set
func debugPrint(format string, args ...interface{}) {
	if os.Getenv("DEBUG") != "" || os.Getenv("VERBOSE") != "" {
		fmt.Printf("DEBUG: "+format, args...)
	}
}

type MCPProcessInterface interface {
	Start() error
	Stop() error
	CallTool(name string, arguments map[string]interface{}) (interface{}, error)
	GetTools() []Tool
}

type Manager struct {
	servers   map[string]*types.MCPServer
	processes map[string]MCPProcessInterface
	storage   *storage.Storage
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	storage := storage.New("")

	m := &Manager{
		servers:   make(map[string]*types.MCPServer),
		processes: make(map[string]MCPProcessInterface),
		storage:   storage,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Load existing servers from storage
	if servers, err := storage.LoadMCPServers(); err == nil {
		m.servers = servers
	}

	go m.healthCheckLoop()
	return m
}

func (m *Manager) AddServer(server *types.MCPServer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if server.ID == "" {
		server.ID = uuid.New().String()
	}

	server.CreatedAt = time.Now()
	server.UpdatedAt = time.Now()
	server.Status = "pending"

	m.servers[server.ID] = server

	// Save to storage
	if err := m.storage.SaveMCPServers(m.servers); err != nil {
		fmt.Printf("Warning: failed to save servers to storage: %v\n", err)
	}

	// For stdio servers, test connection and update status immediately
	if server.Transport == "stdio" {
		m.testStdioServer(server)
	} else {
		// For other transports, connect in background
		go m.connectToServer(server)
	}

	return nil
}

func (m *Manager) RemoveServer(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.servers[id]; !exists {
		return fmt.Errorf("server %s not found", id)
	}

	// Stop the process if it's running
	if process, exists := m.processes[id]; exists {
		process.Stop()
		delete(m.processes, id)
	}

	delete(m.servers, id)

	// Save to storage
	if err := m.storage.SaveMCPServers(m.servers); err != nil {
		fmt.Printf("Warning: failed to save servers to storage: %v\n", err)
	}

	return nil
}

func (m *Manager) GetServer(id string) (*types.MCPServer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, exists := m.servers[id]
	if !exists {
		return nil, fmt.Errorf("server %s not found", id)
	}

	return server, nil
}

func (m *Manager) ListServers() []*types.MCPServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make([]*types.MCPServer, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}

	return servers
}

func (m *Manager) UpdateServerStatus(id, status string) error {
	debugPrint("UpdateServerStatus called for %s with status %s\n", id, status)
	m.mu.Lock()
	defer m.mu.Unlock()

	server, exists := m.servers[id]
	if !exists {
		debugPrint("Server %s not found in UpdateServerStatus\n", id)
		return fmt.Errorf("server %s not found", id)
	}

	debugPrint("Updating server %s status from %s to %s\n", server.Name, server.Status, status)
	server.Status = status
	server.UpdatedAt = time.Now()
	server.LastPing = time.Now()

	debugPrint("Server %s status updated successfully\n", server.Name)
	return nil
}

func (m *Manager) connectToServer(server *types.MCPServer) {
	switch server.Transport {
	case "stdio":
		m.connectStdio(server)
	case "sse":
		m.connectSSE(server)
	case "http":
		m.connectHTTP(server)
	default:
		m.UpdateServerStatus(server.ID, "error")
	}
}

// testStdioServer tests stdio server without mutex complications
func (m *Manager) testStdioServer(server *types.MCPServer) {
	debugPrint("Testing stdio server %s\n", server.Name)

	var process MCPProcessInterface

	// Use mock for testing or specific commands
	if server.URL == "echo" || server.URL == "mock" {
		debugPrint("Creating MockMCPProcess for %s\n", server.Name)
		process = NewMockMCPProcess(server)
	} else {
		debugPrint("Creating MCPProcess for %s\n", server.Name)
		process = NewMCPProcess(server)
	}

	// Test if we can start the process and discover tools
	debugPrint("Starting process for %s\n", server.Name)
	if err := process.Start(); err != nil {
		debugPrint("Failed to start process for %s: %v\n", server.Name, err)
		server.Status = "error"
		return
	}
	debugPrint("Process started successfully for %s\n", server.Name)

	// Get tools and validate the server works
	debugPrint("Getting tools for %s\n", server.Name)
	tools := process.GetTools()
	debugPrint("Got %d tools for %s\n", len(tools), server.Name)

	// If we successfully got tools, the server is working
	if len(tools) > 0 {
		// Store the process
		m.processes[server.ID] = process
		debugPrint("Process stored for %s\n", server.Name)

		// Store tools in capabilities (as requested)
		var capabilities []string
		for _, tool := range tools {
			capabilities = append(capabilities, tool.Name)
		}
		server.Capabilities = capabilities

		// Convert and store detailed tools
		var serverTools []types.Tool
		for _, tool := range tools {
			serverTools = append(serverTools, types.Tool{
				Name:        tool.Name,
				Description: tool.Description,
				Schema:      tool.Schema,
			})
		}
		server.Tools = serverTools

		// Update status directly (we already have the lock from AddServer)
		server.Status = "available"
		server.UpdatedAt = time.Now()
		server.LastPing = time.Now()

		fmt.Printf("Server %s is now available with %d tools\n", server.Name, len(tools))

		// Save to storage
		if err := m.storage.SaveMCPServers(m.servers); err != nil {
			fmt.Printf("Warning: failed to save servers to storage: %v\n", err)
		}
	} else {
		debugPrint("No tools found for %s, marking as error\n", server.Name)
		server.Status = "error"
	}

	debugPrint("stdio server test completed for %s\n", server.Name)
}

func (m *Manager) connectStdio(server *types.MCPServer) {
	// This method is now only used for background connections from connectToServer
	// For immediate stdio testing during AddServer, use testStdioServer instead
	m.testStdioServer(server)
}

func (m *Manager) connectSSE(server *types.MCPServer) {
	m.UpdateServerStatus(server.ID, "connected")
}

func (m *Manager) connectHTTP(server *types.MCPServer) {
	m.UpdateServerStatus(server.ID, "connected")
}

func (m *Manager) healthCheckLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performHealthChecks()
		}
	}
}

func (m *Manager) performHealthChecks() {
	m.mu.RLock()
	servers := make([]*types.MCPServer, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}
	m.mu.RUnlock()

	for _, server := range servers {
		go m.healthCheck(server)
	}
}

func (m *Manager) healthCheck(server *types.MCPServer) {
	if time.Since(server.LastPing) > 60*time.Second {
		m.UpdateServerStatus(server.ID, "unhealthy")
	}
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	// Stop all processes
	for _, process := range m.processes {
		process.Stop()
	}
	m.processes = make(map[string]MCPProcessInterface)
	m.mu.Unlock()

	m.cancel()
}

// CallTool calls a tool on the specified MCP server
func (m *Manager) CallTool(serverID, toolName string, arguments map[string]interface{}) (interface{}, error) {
	m.mu.RLock()
	server, serverExists := m.servers[serverID]
	process, processExists := m.processes[serverID]
	m.mu.RUnlock()

	if !serverExists {
		return nil, fmt.Errorf("MCP server %s not found", serverID)
	}

	// For stdio servers, we might need to start a fresh process for each call
	if server.Transport == "stdio" {
		// For Mock servers, use the existing process
		if server.URL == "echo" || server.URL == "mock" {
			if !processExists {
				return nil, fmt.Errorf("MCP server %s not available", serverID)
			}
			return process.CallTool(toolName, arguments)
		}

		// For real stdio servers, start a fresh process
		var freshProcess MCPProcessInterface
		freshProcess = NewMCPProcess(server)

		if err := freshProcess.Start(); err != nil {
			return nil, fmt.Errorf("failed to start MCP process: %w", err)
		}
		defer freshProcess.Stop()

		return freshProcess.CallTool(toolName, arguments)
	}

	// For other transports (SSE, HTTP), use persistent connection
	if !processExists {
		return nil, fmt.Errorf("MCP server %s not connected", serverID)
	}

	return process.CallTool(toolName, arguments)
}

// GetServerTools returns available tools for a server
func (m *Manager) GetServerTools(serverID string) ([]Tool, error) {
	m.mu.RLock()
	server, serverExists := m.servers[serverID]
	process, processExists := m.processes[serverID]
	m.mu.RUnlock()

	if !serverExists {
		return nil, fmt.Errorf("MCP server %s not found", serverID)
	}

	// For stdio servers, use stored tools if available
	if server.Transport == "stdio" && len(server.Tools) > 0 {
		var tools []Tool
		for _, tool := range server.Tools {
			tools = append(tools, Tool{
				Name:        tool.Name,
				Description: tool.Description,
				Schema:      tool.Schema,
			})
		}
		return tools, nil
	}

	// For other servers or if no stored tools, use process
	if !processExists {
		return nil, fmt.Errorf("MCP server %s not connected", serverID)
	}

	return process.GetTools(), nil
}

// GetAllTools returns all available tools from all connected servers
func (m *Manager) GetAllTools() map[string][]Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	allTools := make(map[string][]Tool)

	// Include all available servers
	for serverID, server := range m.servers {
		if server.Status == "available" || server.Status == "connected" {
			// For stdio servers, use stored tools
			if server.Transport == "stdio" && len(server.Tools) > 0 {
				var tools []Tool
				for _, tool := range server.Tools {
					tools = append(tools, Tool{
						Name:        tool.Name,
						Description: tool.Description,
						Schema:      tool.Schema,
					})
				}
				allTools[server.Name] = tools
			} else if process, exists := m.processes[serverID]; exists {
				// For other servers, use process
				allTools[server.Name] = process.GetTools()
			}
		}
	}

	return allTools
}
