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

type Agent struct {
	mcpManager *mcp.Manager
	llmManager *llm.Manager
}

func New(mcpManager *mcp.Manager, llmManager *llm.Manager) *Agent {
	return &Agent{
		mcpManager: mcpManager,
		llmManager: llmManager,
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

	processedMessage, err := a.processWithLLM(message, provider)
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

	processedMessage, err := a.processWithLLMAndUI(message, provider, display)
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

func (a *Agent) processWithLLM(message string, provider *types.LLMProvider) (string, error) {
	// Get all available MCP tools
	allTools := a.mcpManager.GetAllTools()
	var mcpTools []map[string]interface{}

	for serverName, tools := range allTools {
		for _, tool := range tools {
			// Clean server name for tool naming (remove spaces and special chars)
			cleanServerName := strings.ReplaceAll(strings.ReplaceAll(serverName, " ", "_"), "-", "_")
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

	switch provider.Type {
	case "openai":
		return a.processOpenAIWithMCP(message, provider, mcpTools)
	case "anthropic":
		return a.processAnthropic(message, provider)
	case "google":
		return a.processGoogle(message, provider)
	case "local":
		return a.processLocal(message, provider)
	default:
		return "", fmt.Errorf("unsupported provider type: %s", provider.Type)
	}
}

// processWithLLMAndUI processes message with LLM and UI feedback
func (a *Agent) processWithLLMAndUI(message string, provider *types.LLMProvider, display ui.ToolDisplayInterface) (string, error) {
	// Get all available MCP tools
	allTools := a.mcpManager.GetAllTools()
	var mcpTools []map[string]interface{}

	for serverName, tools := range allTools {
		for _, tool := range tools {
			// Clean server name for tool naming (remove spaces and special chars)
			cleanServerName := strings.ReplaceAll(strings.ReplaceAll(serverName, " ", "_"), "-", "_")
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

	switch provider.Type {
	case "openai":
		return a.processOpenAIWithMCPAndUI(message, provider, mcpTools, display)
	case "anthropic":
		return a.processAnthropic(message, provider)
	case "google":
		return a.processGoogle(message, provider)
	case "local":
		return a.processLocal(message, provider)
	default:
		return "", fmt.Errorf("unsupported provider type: %s", provider.Type)
	}
}

func (a *Agent) processOpenAI(message string, provider *types.LLMProvider) (string, error) {
	return llm.CallOpenAI(provider, message)
}

func (a *Agent) processOpenAIWithMCP(message string, provider *types.LLMProvider, mcpTools []map[string]interface{}) (string, error) {
	// Enhance message with tool availability context
	enhancedMessage := message
	if len(mcpTools) > 0 {
		enhancedMessage = fmt.Sprintf(`%s

Available tools: %d desktop tools including file operations, system commands, and process management.
Use multiple tools as needed to provide comprehensive answers.`, message, len(mcpTools))
	}

	// Convert MCP tools to OpenAI format
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
				return nil, fmt.Errorf("server %s not found", serverName)
			}
		}
		return nil, fmt.Errorf("tool %s not found", name)
	}

	// If no tools available, fall back to regular call
	if len(tools) == 0 {
		return llm.CallOpenAI(provider, enhancedMessage)
	}

	return llm.CallOpenAIWithTools(provider, enhancedMessage, tools, toolCaller)
}

// processOpenAIWithMCPAndUI processes OpenAI with MCP tools and UI feedback
func (a *Agent) processOpenAIWithMCPAndUI(message string, provider *types.LLMProvider, mcpTools []map[string]interface{}, display ui.ToolDisplayInterface) (string, error) {
	// Enhance message with tool availability context
	enhancedMessage := message
	if len(mcpTools) > 0 {
		enhancedMessage = fmt.Sprintf(`%s

Available tools: %d desktop tools including file operations, system commands, and process management.
Use multiple tools as needed to provide comprehensive answers.`, message, len(mcpTools))
	}

	// Convert MCP tools to OpenAI format
	tools := llm.ConvertMCPToolsToOpenAI(mcpTools)

	// Create execution summary tracker
	summary := ui.ExecutionSummary{
		ToolCalls: []ui.ToolCallRecord{},
	}

	// Track user control state
	var userAborted bool
	var autoApproveAll bool

	// Create tool caller function with UI feedback
	toolCaller := func(name string, args map[string]interface{}) (interface{}, error) {
		// Check abort status first
		if userAborted {
			return nil, fmt.Errorf("execution aborted by user - no further tools will be executed")
		}

		// Check auto-approve status
		if autoApproveAll {
			display.ShowProgress("Auto-approving tool due to 'auto-approve all' selection")
			// Continue with normal execution (don't return early)
		}
		// Find the corresponding MCP tool
		for _, mcpTool := range mcpTools {
			if mcpTool["name"] == name {
				serverName := mcpTool["serverName"].(string)
				toolName := mcpTool["toolName"].(string)

				// Show tool call to user
				display.ShowToolCall(serverName, toolName, args)

				// In interactive mode, ask for approval (unless auto-approve is active)
				var approved bool
				if autoApproveAll {
					// Auto-approve mode: don't ask, just approve
					approved = true
					display.ShowProgress("Auto-approving tool execution")
				} else {
					// Normal mode: ask for approval
					var err error
					approved, err = display.PromptToolApproval(serverName, toolName, args)
					if err != nil {
						if err.Error() == "AUTO_APPROVE_ALL" {
							autoApproveAll = true
							display.ShowProgress("Auto-approve mode activated - remaining tools will be executed automatically")
							approved = true // Approve this tool too
						} else if err.Error() == "ABORT" {
							userAborted = true
							display.ShowError(fmt.Errorf("execution aborted by user"))
							return nil, fmt.Errorf("execution aborted by user - session terminated")
						} else {
							return nil, err
						}
					}

					if !approved {
						display.ShowProgress("Tool execution skipped by user")
						return "Tool execution skipped by user", nil
					}
				}

				// Record start time
				startTime := time.Now()

				// Find server ID by name and execute tool
				servers := a.mcpManager.ListServers()
				for _, server := range servers {
					if server.Name == serverName {
						result, err := a.mcpManager.CallTool(server.ID, toolName, args)
						duration := time.Since(startTime)

						// Record tool call
						record := ui.ToolCallRecord{
							ServerName: serverName,
							ToolName:   toolName,
							Duration:   duration,
							Success:    err == nil,
						}
						if err != nil {
							record.Error = err.Error()
							display.ShowError(err)
						} else {
							display.ShowToolResult(result, duration)
						}
						summary.ToolCalls = append(summary.ToolCalls, record)
						summary.TotalTools++
						if err == nil {
							summary.SuccessfulCalls++
						} else {
							summary.FailedCalls++
						}
						summary.TotalDuration += duration

						return result, err
					}
				}
				return nil, fmt.Errorf("server %s not found", serverName)
			}
		}
		return nil, fmt.Errorf("tool %s not found", name)
	}

	// If no tools available, fall back to regular call
	if len(tools) == 0 {
		display.ShowProgress("Processing with LLM (no tools available)...")
		return llm.CallOpenAI(provider, enhancedMessage)
	}

	display.ShowProgress("Processing with LLM and available tools...")
	result, err := llm.CallOpenAIWithTools(provider, enhancedMessage, tools, toolCaller)

	// Show execution summary if tools were used
	if summary.TotalTools > 0 {
		display.ShowSummary(summary)
	}

	return result, err
}

func (a *Agent) processAnthropic(message string, provider *types.LLMProvider) (string, error) {
	return llm.CallAnthropic(provider, message)
}

func (a *Agent) processGoogle(message string, provider *types.LLMProvider) (string, error) {
	// Google API implementation would go here
	return fmt.Sprintf("[Google %s] Processed: %s", provider.Model, message), nil
}

func (a *Agent) processLocal(message string, provider *types.LLMProvider) (string, error) {
	return llm.CallLocal(provider, message)
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
