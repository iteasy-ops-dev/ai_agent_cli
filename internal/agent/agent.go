package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/iteasy-ops-dev/syseng-agent/internal/llm"
	"github.com/iteasy-ops-dev/syseng-agent/internal/mcp"
	"github.com/iteasy-ops-dev/syseng-agent/internal/ui"
	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
)

// StreamResponse represents a streaming response chunk
type StreamResponse struct {
	Content string
	Error   string
	Done    bool
}

type Agent struct {
	mcpManager       *mcp.Manager
	llmManager       *llm.Manager
	processorFactory llm.ProcessorFactory
}

func New(mcpManager *mcp.Manager, llmManager *llm.Manager) *Agent {
	return &Agent{
		mcpManager:       mcpManager,
		llmManager:       llmManager,
		processorFactory: llm.NewDefaultProcessorFactory(),
	}
}

func (a *Agent) ProcessRequest(message, mcpServerID, providerID string) (*types.AgentResponse, error) {
	request := &types.AgentRequest{
		ID:          uuid.New().String(),
		Message:     message,
		MCPServerID: mcpServerID,
		ProviderID:  providerID,
		CreatedAt:   time.Now(),
	}

	response := &types.AgentResponse{
		ID:        uuid.New().String(),
		RequestID: request.ID,
		CreatedAt: time.Now(),
	}

	var provider *types.LLMProvider
	var err error

	if providerID != "" {
		provider, err = a.llmManager.GetProvider(providerID)
	} else {
		provider, err = a.llmManager.GetActiveProvider()
	}

	if err != nil {
		response.Error = fmt.Sprintf("Provider error: %v", err)
		return response, nil
	}

	// Create processor for this provider
	processor, err := a.processorFactory.CreateProcessor(provider)
	if err != nil {
		response.Error = fmt.Sprintf("Failed to create processor: %v", err)
		return response, nil
	}

	// Process with tools (no MCP tools for simple request)
	processedMessage, err := processor.ProcessWithTools(message, []llm.Tool{}, nil)
	if err != nil {
		response.Error = fmt.Sprintf("LLM processing error: %v", err)
		return response, nil
	}

	if mcpServerID != "" {
		mcpData, err := a.processWithMCP(processedMessage, mcpServerID)
		if err != nil {
			response.Error = fmt.Sprintf("MCP processing error: %v", err)
			return response, nil
		}
		response.Data = mcpData
	}

	response.Message = processedMessage
	return response, nil
}

// ProcessRequestWithUI processes a request with enhanced UI feedback
// ProcessConversationWithStreaming processes a conversation with streaming support
func (a *Agent) ProcessConversationWithStreaming(session *types.ConversationSession, message string, display ui.ToolDisplayInterface) (<-chan StreamResponse, error) {
	ch := make(chan StreamResponse, 10)
	
	go func() {
		defer close(ch)
		
		if display != nil {
			display.ShowProgress("Initializing AI agent...")
		}

		// Add user message to conversation
		session.AddMessage("user", message)



		var provider *types.LLMProvider
		var err error

		if session.ProviderID != "" {
			provider, err = a.llmManager.GetProvider(session.ProviderID)
		} else {
			provider, err = a.llmManager.GetActiveProvider()
		}

		if err != nil {
			ch <- StreamResponse{Error: fmt.Sprintf("Provider error: %v", err)}
			return
		}

		if display != nil {
			display.ShowProgress("Finding LLM provider...")
		}

		// Create processor for this provider
		processor, err := a.processorFactory.CreateProcessor(provider)
		if err != nil {
			ch <- StreamResponse{Error: fmt.Sprintf("Failed to create processor: %v", err)}
			return
		}

		// Prepare MCP tools and tool caller
		tools, toolCaller := a.prepareMCPTools(session.MCPServerID, display)

		// Process conversation with streaming support
		err = a.processConversationWithToolsStreaming(processor, session, tools, toolCaller, display, ch)
		if err != nil {
			ch <- StreamResponse{Error: fmt.Sprintf("Processing error: %v", err)}
			return
		}
	}()
	
	return ch, nil
}

// ProcessConversation processes a message within a conversation context using the new processor system
func (a *Agent) ProcessConversation(session *types.ConversationSession, message string, display ui.ToolDisplayInterface) (*types.AgentResponse, error) {
	if display != nil {
		display.ShowProgress("Initializing AI agent...")
	}

	// Add user message to conversation
	session.AddMessage("user", message)

	request := &types.AgentRequest{
		ID:          uuid.New().String(),
		Message:     message,
		MCPServerID: session.MCPServerID,
		ProviderID:  session.ProviderID,
		CreatedAt:   time.Now(),
	}

	response := &types.AgentResponse{
		ID:        uuid.New().String(),
		RequestID: request.ID,
		CreatedAt: time.Now(),
	}

	var provider *types.LLMProvider
	var err error

	if session.ProviderID != "" {
		provider, err = a.llmManager.GetProvider(session.ProviderID)
	} else {
		provider, err = a.llmManager.GetActiveProvider()
	}

	if err != nil {
		response.Error = fmt.Sprintf("Provider error: %v", err)
		return response, nil
	}

	if display != nil {
		display.ShowProgress("Finding LLM provider...")
	}

	// Create processor for this provider
	processor, err := a.processorFactory.CreateProcessor(provider)
	if err != nil {
		response.Error = fmt.Sprintf("Failed to create processor: %v", err)
		return response, nil
	}

	// Prepare MCP tools and tool caller
	tools, toolCaller := a.prepareMCPTools(session.MCPServerID, display)

	// Process conversation with UI feedback
	result, err := processor.ProcessConversationWithUI(session, tools, toolCaller, display)
	if err != nil {
		response.Error = fmt.Sprintf("Processing error: %v", err)
		return response, nil
	}

	// Add assistant response to conversation
	session.AddMessage("assistant", result)

	response.Message = result
	return response, nil
}

// prepareMCPTools prepares MCP tools and creates a tool caller function
func (a *Agent) prepareMCPTools(mcpServerID string, display ui.ToolDisplayInterface) ([]llm.Tool, llm.ToolCaller) {
	if display != nil {
		display.ShowProgress("Loading tools from MCP servers...")
	}

	// Get all available tools from MCP servers
	allMCPTools := a.mcpManager.GetAllTools()
	
	var mcpTools []map[string]interface{}
	
	for serverName, tools := range allMCPTools {
		// Clean server name for tool naming
		cleanServerName := strings.ReplaceAll(serverName, " ", "_")
		cleanServerName = strings.ReplaceAll(cleanServerName, "-", "_")
		
		for _, tool := range tools {
			mcpTool := map[string]interface{}{
				"name":        fmt.Sprintf("%s_%s", cleanServerName, tool.Name),
				"description": fmt.Sprintf("[%s] %s", serverName, tool.Description),
				"inputSchema": tool.Schema,
				"serverName":  serverName,
				"toolName":    tool.Name,
			}
			mcpTools = append(mcpTools, mcpTool)
		}
	}

	if display != nil {
		display.ShowProgress(fmt.Sprintf("✅ Loaded %d tools total", len(mcpTools)))
	}

	// Convert MCP tools to LLM format
	tools := llm.ConvertMCPToolsToOpenAI(mcpTools)

	// Create tool caller function
	toolCaller := func(name string, args map[string]interface{}) (interface{}, error) {
		// Find the corresponding MCP tool
		for _, mcpTool := range mcpTools {
			if mcpTool["name"] == name {
				serverName := mcpTool["serverName"].(string)
				toolName := mcpTool["toolName"].(string)
				
				// Find server ID by name
				servers := a.mcpManager.ListServers()
				for _, server := range servers {
					if server.Name == serverName {
						return a.mcpManager.CallTool(server.ID, toolName, args)
					}
				}
			}
		}
		return nil, fmt.Errorf("tool %s not found", name)
	}

	return tools, toolCaller
}

func (a *Agent) ProcessRequestWithUI(message, mcpServerID, providerID string, interactive bool) (*types.AgentResponse, error) {
	// Create appropriate display interface with enhancements
	var display ui.ToolDisplayInterface
	if interactive {
		base := ui.NewInteractiveDisplay()
		display = ui.NewTimedProgressDisplay(ui.NewSpinnerDisplay(base))
	} else {
		base := ui.NewNonInteractiveDisplay()
		display = ui.NewTimedProgressDisplay(ui.NewSpinnerDisplay(base))
	}

	display.ShowProgress("Initializing AI agent...")

	request := &types.AgentRequest{
		ID:          uuid.New().String(),
		Message:     message,
		MCPServerID: mcpServerID,
		ProviderID:  providerID,
		CreatedAt:   time.Now(),
	}

	response := &types.AgentResponse{
		ID:        uuid.New().String(),
		RequestID: request.ID,
		CreatedAt: time.Now(),
	}

	var provider *types.LLMProvider
	var err error

	display.ShowProgress("Finding LLM provider...")

	if providerID != "" {
		provider, err = a.llmManager.GetProvider(providerID)
	} else {
		provider, err = a.llmManager.GetActiveProvider()
	}

	if err != nil {
		display.ShowError(fmt.Errorf("Provider error: %v", err))
		response.Error = fmt.Sprintf("Provider error: %v", err)
		return response, nil
	}

	// Create processor for this provider
	processor, err := a.processorFactory.CreateProcessor(provider)
	if err != nil {
		display.ShowError(fmt.Errorf("Failed to create processor: %v", err))
		response.Error = fmt.Sprintf("Failed to create processor: %v", err)
		return response, nil
	}

	// Prepare MCP tools and tool caller
	tools, toolCaller := a.prepareMCPTools(mcpServerID, display)

	// Process with UI feedback
	processedMessage, err := processor.ProcessWithUI(message, tools, toolCaller, display)
	if err != nil {
		display.ShowError(fmt.Errorf("LLM processing error: %v", err))
		response.Error = fmt.Sprintf("LLM processing error: %v", err)
		return response, nil
	}

	if mcpServerID != "" {
		display.ShowProgress("Processing with MCP server...")
		mcpData, err := a.processWithMCP(processedMessage, mcpServerID)
		if err != nil {
			display.ShowError(fmt.Errorf("MCP processing error: %v", err))
			response.Error = fmt.Sprintf("MCP processing error: %v", err)
			return response, nil
		}
		response.Data = mcpData
	}

	response.Message = processedMessage
	return response, nil
}




// processConversationWithToolsStreaming handles conversation with tools and streaming
func (a *Agent) processConversationWithToolsStreaming(processor llm.LLMProcessor, session *types.ConversationSession, tools []llm.Tool, toolCaller llm.ToolCaller, display ui.ToolDisplayInterface, ch chan<- StreamResponse) error {
	// For now, we'll use the existing tool processing logic and stream the final result
	// This ensures tools work correctly while providing streaming output
	result, err := processor.ProcessConversationWithUI(session, tools, toolCaller, display)
	if err != nil {
		return err
	}
	
	// Add assistant response to conversation
	session.AddMessage("assistant", result)
	
	// Stream the final result character by character for streaming effect
	for _, char := range result {
		ch <- StreamResponse{Content: string(char)}
		// Small delay to simulate streaming effect
		time.Sleep(10 * time.Millisecond)
	}
	ch <- StreamResponse{Done: true}
	
	return nil
}

func (a *Agent) processWithMCP(message, serverID string) (map[string]interface{}, error) {
	server, err := a.mcpManager.GetServer(serverID)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"server_id":   server.ID,
		"server_name": server.Name,
		"status":      server.Status,
		"processed":   true,
	}

	return data, nil
}

func (a *Agent) StartServer(port string) error {
	r := mux.NewRouter()

	r.HandleFunc("/api/v1/query", a.handleQuery).Methods("POST")
	r.HandleFunc("/api/v1/health", a.handleHealth).Methods("GET")
	r.HandleFunc("/api/v1/mcp/servers", a.handleMCPServers).Methods("GET")
	r.HandleFunc("/api/v1/llm/providers", a.handleLLMProviders).Methods("GET")

	addr := ":" + port
	log.Printf("Server starting on %s", addr)

	return http.ListenAndServe(addr, r)
}

func (a *Agent) handleQuery(w http.ResponseWriter, r *http.Request) {
	var request types.AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := a.ProcessRequest(request.Message, request.MCPServerID, request.ProviderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (a *Agent) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (a *Agent) handleMCPServers(w http.ResponseWriter, r *http.Request) {
	servers := a.mcpManager.ListServers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servers)
}

func (a *Agent) handleLLMProviders(w http.ResponseWriter, r *http.Request) {
	providers := a.llmManager.ListProviders()

	for _, provider := range providers {
		provider.APIKey = "***"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

