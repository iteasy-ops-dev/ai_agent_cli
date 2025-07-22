package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
)

// LocalClient implements FullClient for local LLM models (like Ollama with OpenAI compatibility)
type LocalClient struct {
	provider   *types.LLMProvider
	httpClient *http.Client // Keep http.Client for now due to complex custom logic
}

// NewLocalClient creates a new local LLM client
func NewLocalClient(provider *types.LLMProvider) *LocalClient {
	return &LocalClient{
		provider: provider,
		httpClient: &http.Client{
			Timeout: LocalModelTimeout, // Local models might be slower
		},
	}
}

// ProcessMessage processes a simple message
func (c *LocalClient) ProcessMessage(message string) (string, error) {
	if c.provider.Endpoint == "" {
		return "", fmt.Errorf(ErrEndpointRequired)
	}

	// Use OpenAI-compatible format for local providers
	reqBody := OpenAIRequest{
		Model: c.provider.Model,
		Messages: []Message{
			{Role: RoleUser, Content: message},
		},
	}

	return c.executeRequest(reqBody)
}

// ProcessWithTools processes a message with available tools
func (c *LocalClient) ProcessWithTools(message string, tools []Tool, toolCaller ToolCaller) (string, error) {
	if c.provider.Endpoint == "" {
		return "", fmt.Errorf(ErrEndpointRequired)
	}

	// Enhanced message with tool context for local models
	enhancedMessage := message
	if len(tools) > 0 {
		debugPrint("Providing %d tools to local LLM\n", len(tools))

		toolContext := "\n\nIMPORTANT: Available tools for use (use EXACT names):\n"
		for _, tool := range tools {
			debugPrint("Tool available: %s - %s\n", tool.Function.Name, tool.Function.Description)
			toolContext += fmt.Sprintf("- %s: %s\n", tool.Function.Name, tool.Function.Description)
		}
		toolContext += "\nCRITICAL RULES:\n"
		toolContext += "1. ONLY use tools from the list above with EXACT names\n"
		toolContext += "2. For system commands, use: Desktop_Commander_start_process\n"
		toolContext += "3. For file operations, use: Desktop_Commander_read_file or Desktop_Commander_list_directory\n"
		toolContext += "4. Tool format: {\"use_tool\": \"EXACT_TOOL_NAME\", \"parameters\": {\"key\": \"value\"}}\n"
		toolContext += "5. If no suitable tool exists, respond normally without tools\n"
		toolContext += "\nExample: {\"use_tool\": \"Desktop_Commander_start_process\", \"parameters\": {\"command\": \"hostname\"}}\n"

		enhancedMessage = toolContext + "\n\n" + message
		debugPrint("Enhanced message length: %d characters\n", len(enhancedMessage))
	}

	// Start with user message
	messages := []Message{
		{Role: RoleUser, Content: enhancedMessage},
	}

	// Simple iteration for local models (limit to avoid overwhelming local models)
	for iteration := 0; iteration < LocalMaxToolIterations; iteration++ {
		reqBody := OpenAIRequest{
			Model:    c.provider.Model,
			Messages: messages,
		}

		// Local models might support OpenAI-compatible tools
		if len(tools) > 0 && c.supportsNativeTools() {
			reqBody.Tools = tools
			reqBody.ToolChoice = ToolChoiceAuto
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return "", fmt.Errorf(ErrFailedToMarshal, err)
		}

		req, err := http.NewRequest("POST", c.provider.Endpoint, bytes.NewBuffer(jsonData))
		if err != nil {
			return "", fmt.Errorf(ErrFailedToCreateRequest, err)
		}

		req.Header.Set(HeaderContentType, ContentTypeJSON)
		if c.provider.APIKey != "" {
			req.Header.Set(HeaderAuthorization, AuthBearerPrefix+c.provider.APIKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf(ErrFailedToSendRequest, err)
		}

		// Read response body for debugging
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf(ErrFailedToReadResponse, err)
		}

		debugPrint("Response status: %d\n", resp.StatusCode)
		debugPrint("Response body: %s\n", string(body))

		if resp.StatusCode != StatusOK {
			return "", fmt.Errorf(ErrAPIRequestFailed, resp.StatusCode, string(body))
		}

		// Try to parse using multi-format logic
		isOllamaAPI := c.isOllamaAPI()
		if isOllamaAPI {
			// For Ollama, we don't expect tool calls in the response format
			// Just get the content and check for JSON tool usage pattern
			content, err := c.parseResponse(body, true)
			if err != nil {
				return "", fmt.Errorf("failed to parse Ollama response: %v", err)
			}

			// Create a mock choice for compatibility with the rest of the logic
			choice := Choice{
				Message: Message{
					Role:    RoleAssistant,
					Content: content,
				},
			}

			messages = append(messages, choice.Message)

			// Check for JSON tool usage pattern in response content
			response := choice.Message.Content
			if toolCaller != nil && len(tools) > 0 {
				if toolCall := c.parseToolCallFromResponse(response); toolCall != nil {
					debugPrint("Local LLM (Ollama) requested tool: %s with params: %v\n", toolCall.ToolName, toolCall.Parameters)

					// Validate tool exists in available tools
					toolExists := false
					for _, tool := range tools {
						if tool.Function.Name == toolCall.ToolName {
							toolExists = true
							break
						}
					}

					if !toolExists {
						debugPrint("Tool not found: %s\n", toolCall.ToolName)
						var availableTools []string
						for _, tool := range tools {
							availableTools = append(availableTools, tool.Function.Name)
						}
						errorMsg := fmt.Sprintf("ERROR: Tool '%s' not found. Available tools: %s. Please use EXACT tool names from the list.",
							toolCall.ToolName, strings.Join(availableTools, ", "))
						messages = append(messages, Message{
							Role:    RoleUser,
							Content: errorMsg,
						})
						continue
					}

					result, err := toolCaller(toolCall.ToolName, toolCall.Parameters)
					if err != nil {
						errorMsg := fmt.Sprintf("Tool execution failed: %v. Available tools: ", err)
						for _, tool := range tools {
							errorMsg += tool.Function.Name + ", "
						}
						errorMsg += "Please use correct tool names and try again."
						messages = append(messages, Message{
							Role:    RoleUser,
							Content: errorMsg,
						})
						continue
					}

					var resultStr string
					switch v := result.(type) {
					case string:
						resultStr = v
					case map[string]interface{}:
						if jsonBytes, err := json.Marshal(v); err == nil {
							resultStr = string(jsonBytes)
						} else {
							resultStr = fmt.Sprintf("%v", v)
						}
					default:
						resultStr = fmt.Sprintf("%v", v)
					}

					messages = append(messages, Message{
						Role:    RoleUser,
						Content: fmt.Sprintf("Tool '%s' result: %s\n\nPlease provide a final response based on this information.", toolCall.ToolName, resultStr),
					})
					continue
				}
			}

			// No tool calls, return the response
			return response, nil
		} else {
			// Try OpenAI format parsing
			var openAIResp OpenAIResponse
			if err := json.Unmarshal(body, &openAIResp); err != nil {
				return "", fmt.Errorf("failed to decode OpenAI response: %v", err)
			}

			if len(openAIResp.Choices) == 0 {
				return "", fmt.Errorf(ErrNoResponseChoices)
			}

			choice := openAIResp.Choices[0]
			messages = append(messages, choice.Message)

			// Check for native tool calls first
			if len(choice.Message.ToolCalls) > 0 && toolCaller != nil {
				// Handle native tool calls like OpenAI
				for _, toolCall := range choice.Message.ToolCalls {
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
						messages = append(messages, Message{
							Role:       "tool",
							Content:    fmt.Sprintf("Error parsing arguments: %v", err),
							ToolCallID: toolCall.ID,
							Name:       toolCall.Function.Name,
						})
						continue
					}

					result, err := toolCaller(toolCall.Function.Name, args)
					if err != nil {
						messages = append(messages, Message{
							Role:       "tool",
							Content:    fmt.Sprintf("Error: %v", err),
							ToolCallID: toolCall.ID,
							Name:       toolCall.Function.Name,
						})
						continue
					}

					var resultStr string
					switch v := result.(type) {
					case string:
						resultStr = v
					case map[string]interface{}:
						if jsonBytes, err := json.Marshal(v); err == nil {
							resultStr = string(jsonBytes)
						} else {
							resultStr = fmt.Sprintf("%v", v)
						}
					default:
						resultStr = fmt.Sprintf("%v", v)
					}

					messages = append(messages, Message{
						Role:       RoleTool,
						Content:    resultStr,
						ToolCallID: toolCall.ID,
						Name:       toolCall.Function.Name,
					})
				}
				continue
			}

			// Check for JSON tool usage pattern in response content
			response := choice.Message.Content
			if toolCaller != nil && len(tools) > 0 {
				if toolCall := c.parseToolCallFromResponse(response); toolCall != nil {
					debugPrint("Local LLM (OpenAI format) requested tool: %s with params: %v\n", toolCall.ToolName, toolCall.Parameters)

					// Validate tool exists in available tools
					toolExists := false
					for _, tool := range tools {
						if tool.Function.Name == toolCall.ToolName {
							toolExists = true
							break
						}
					}

					if !toolExists {
						debugPrint("Tool not found: %s\n", toolCall.ToolName)
						var availableTools []string
						for _, tool := range tools {
							availableTools = append(availableTools, tool.Function.Name)
						}
						errorMsg := fmt.Sprintf("ERROR: Tool '%s' not found. Available tools: %s. Please use EXACT tool names from the list.",
							toolCall.ToolName, strings.Join(availableTools, ", "))
						messages = append(messages, Message{
							Role:    RoleUser,
							Content: errorMsg,
						})
						continue
					}

					result, err := toolCaller(toolCall.ToolName, toolCall.Parameters)
					if err != nil {
						errorMsg := fmt.Sprintf("Tool execution failed: %v. Available tools: ", err)
						for _, tool := range tools {
							errorMsg += tool.Function.Name + ", "
						}
						errorMsg += "Please use correct tool names and try again."
						messages = append(messages, Message{
							Role:    RoleUser,
							Content: errorMsg,
						})
						continue
					}

					// Add tool result to conversation
					var resultStr string
					switch v := result.(type) {
					case string:
						resultStr = v
					case map[string]interface{}:
						if jsonBytes, err := json.Marshal(v); err == nil {
							resultStr = string(jsonBytes)
						} else {
							resultStr = fmt.Sprintf("%v", v)
						}
					default:
						resultStr = fmt.Sprintf("%v", v)
					}

					messages = append(messages, Message{
						Role:    RoleUser,
						Content: fmt.Sprintf("Tool '%s' result: %s\n\nPlease provide a final response based on this information.", toolCall.ToolName, resultStr),
					})
					continue
				}
			}

			// No tool calls, return the response
			return response, nil
		}
	}

	return "", fmt.Errorf(ErrMaxIterationsExceeded, LocalMaxToolIterations)
}

// ToolCallRequest represents a tool call parsed from local model response
type ToolCallRequest struct {
	ToolName   string                 `json:"use_tool"`
	Parameters map[string]interface{} `json:"parameters"`
}

// OllamaResponse represents the native Ollama API response format
type OllamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

// OllamaRequest represents the native Ollama API request format
type OllamaRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Stream bool `json:"stream"`
}

// debugPrint prints debug messages only if DEBUG environment variable is set
func debugPrint(format string, args ...interface{}) {
	if os.Getenv(EnvDebug) != "" || os.Getenv(EnvVerbose) != "" {
		fmt.Printf("DEBUG LocalClient: "+format, args...)
	}
}

// parseToolCallFromResponse tries to parse tool call JSON from the response
func (c *LocalClient) parseToolCallFromResponse(response string) *ToolCallRequest {
	// Look for JSON pattern in the response
	start := strings.Index(response, LocalToolPattern)
	if start == -1 {
		return nil
	}

	// Find the end of the JSON object
	braceCount := 0
	end := start
	for i := start; i < len(response); i++ {
		if response[i] == '{' {
			braceCount++
		} else if response[i] == '}' {
			braceCount--
			if braceCount == 0 {
				end = i + 1
				break
			}
		}
	}

	if braceCount != 0 {
		return nil
	}

	jsonStr := response[start:end]
	var toolCall ToolCallRequest
	if err := json.Unmarshal([]byte(jsonStr), &toolCall); err != nil {
		return nil
	}

	if toolCall.ToolName == "" {
		return nil
	}

	return &toolCall
}

// ProcessConversation processes a message within conversation context
func (c *LocalClient) ProcessConversation(session *types.ConversationSession) (string, error) {
	// Convert conversation to simple messages (keep it lightweight for local models)
	messages := c.convertConversationToLocal(session, LocalContextWindowSize) // Keep last few messages only

	reqBody := OpenAIRequest{
		Model:    c.provider.Model,
		Messages: messages,
	}

	return c.executeRequest(reqBody)
}

// ProcessConversationWithTools processes conversation with tools
func (c *LocalClient) ProcessConversationWithTools(session *types.ConversationSession, tools []Tool, toolCaller ToolCaller) (string, error) {
	// Get last user message and add tool context
	lastMessage := session.GetLastUserMessage()

	return c.ProcessWithTools(lastMessage, tools, toolCaller)
}

// convertConversationToLocal converts conversation for local models (simplified)
func (c *LocalClient) convertConversationToLocal(session *types.ConversationSession, maxMessages int) []Message {
	var messages []Message

	// Start from the end and take only recent messages
	startIdx := len(session.Messages) - maxMessages
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(session.Messages); i++ {
		msg := session.Messages[i]
		if msg.Role == "system" {
			continue // Skip system messages for simplicity
		}

		messages = append(messages, Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return messages
}

// executeRequest executes a simple request without tools
func (c *LocalClient) executeRequest(reqBody OpenAIRequest) (string, error) {
	if c.provider.Endpoint == "" {
		return "", fmt.Errorf(ErrEndpointRequired)
	}

	// Determine API format and prepare request accordingly
	isOllamaAPI := c.isOllamaAPI()
	debugPrint("API format detected: isOllama=%v, endpoint=%s\n", isOllamaAPI, c.provider.Endpoint)

	var jsonData []byte
	var err error

	if isOllamaAPI {
		// Convert OpenAI format to Ollama format
		ollamaReq := OllamaRequest{
			Model:  reqBody.Model,
			Stream: false, // Explicitly disable streaming
		}

		for _, msg := range reqBody.Messages {
			ollamaReq.Messages = append(ollamaReq.Messages, struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}

		jsonData, err = json.Marshal(ollamaReq)
		debugPrint("Using Ollama format: %s\n", string(jsonData))
	} else {
		// Use OpenAI format
		jsonData, err = json.Marshal(reqBody)
		debugPrint("Using OpenAI format: %s\n", string(jsonData))
	}

	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", c.provider.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.provider.APIKey)
	}

	debugPrint("Making request to %s\n", c.provider.Endpoint)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	debugPrint("Response status: %d\n", resp.StatusCode)
	debugPrint("Response body: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Try to parse response in multiple formats
	content, err := c.parseResponse(body, isOllamaAPI)
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	return content, nil
}

// isOllamaAPI determines if this is an Ollama API endpoint
func (c *LocalClient) isOllamaAPI() bool {
	endpoint := strings.ToLower(c.provider.Endpoint)

	// Check for typical Ollama endpoints
	if strings.Contains(endpoint, OllamaPortPattern) ||
		strings.Contains(endpoint, OllamaNamePattern) ||
		strings.HasSuffix(endpoint, OllamaAPIPath) {
		return true
	}

	// Default to OpenAI-compatible format
	return false
}

// parseResponse tries to parse the response in different formats
func (c *LocalClient) parseResponse(body []byte, isOllamaAPI bool) (string, error) {
	if isOllamaAPI {
		// Handle Ollama streaming response (multiple JSON objects separated by newlines)
		content, err := c.parseOllamaStreamingResponse(body)
		if err == nil {
			debugPrint("Successfully parsed as Ollama streaming response\n")
			return content, nil
		}
		debugPrint("Failed to parse as Ollama streaming format: %v\n", err)

		// Try single Ollama response format
		var ollamaResp OllamaResponse
		if err := json.Unmarshal(body, &ollamaResp); err == nil {
			debugPrint("Successfully parsed as single Ollama response\n")
			return ollamaResp.Message.Content, nil
		}
		debugPrint("Failed to parse as single Ollama format, trying OpenAI format\n")
	}

	// Try OpenAI format
	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err == nil {
		debugPrint("Successfully parsed as OpenAI response\n")
		if len(openAIResp.Choices) == 0 {
			return "", fmt.Errorf("no response choices received")
		}
		return openAIResp.Choices[0].Message.Content, nil
	}

	// Try to extract content from any JSON response
	var genericResp map[string]interface{}
	if err := json.Unmarshal(body, &genericResp); err == nil {
		debugPrint("Parsing as generic JSON response\n")

		// Try various possible content fields
		contentFields := []string{"content", "text", "response", "message", "output"}
		for _, field := range contentFields {
			if content, ok := genericResp[field]; ok {
				if contentStr, ok := content.(string); ok {
					debugPrint("Found content in field '%s'\n", field)
					return contentStr, nil
				}
			}
		}

		// Try nested message.content
		if message, ok := genericResp["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].(string); ok {
				debugPrint("Found content in message.content\n")
				return content, nil
			}
		}
	}

	return "", fmt.Errorf("unable to parse response body as any known format: %s", string(body))
}

// parseOllamaStreamingResponse parses Ollama streaming response format
func (c *LocalClient) parseOllamaStreamingResponse(body []byte) (string, error) {
	bodyStr := string(body)
	lines := strings.Split(bodyStr, "\n")

	var completeContent strings.Builder

	debugPrint("Parsing %d lines from Ollama streaming response\n", len(lines))

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var resp OllamaResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			debugPrint("Failed to parse line %d as JSON: %v\n", i, err)
			continue
		}

		// Accumulate content from each chunk
		if resp.Message.Content != "" {
			completeContent.WriteString(resp.Message.Content)
		}

		// If this is the final chunk (done=true), we can stop
		if resp.Done {
			debugPrint("Found final chunk with done=true\n")
			break
		}
	}

	content := completeContent.String()
	if content == "" {
		return "", fmt.Errorf("no content found in streaming response")
	}

	debugPrint("Assembled complete content: %s\n", content)
	return content, nil
}

// supportsNativeTools checks if this local provider supports OpenAI-style function calling
func (c *LocalClient) supportsNativeTools() bool {
	// Most local models don't support native function calling yet
	// This could be made configurable in the future
	return false
}

// Interface compliance methods

func (c *LocalClient) GetProviderInfo() ProviderInfo {
	return ProviderInfo{
		Type:     c.provider.Type,
		Model:    c.provider.Model,
		Endpoint: c.provider.Endpoint,
		Version:  "local",
	}
}

func (c *LocalClient) IsHealthy() bool {
	return c.provider.Endpoint != ""
}

func (c *LocalClient) SupportsFunctionCalling() bool {
	// Limited support through JSON parsing
	return true
}

func (c *LocalClient) SupportsConversation() bool {
	return true
}

func (c *LocalClient) GetCapabilities() ClientCapabilities {
	return ClientCapabilities{
		SupportsTools:        true, // Through JSON parsing
		SupportsConversation: true,
		SupportsStreaming:    true, // Ollama supports streaming
		MaxTokens:            getMaxTokensForLocalModel(c.provider.Model),
		MaxConversationTurn:  LocalMaxConversationTurns, // Keep it low for local models
	}
}

// ProcessMessageStream processes a message and streams the response
func (c *LocalClient) ProcessMessageStream(message string) (<-chan StreamChunk, error) {
	if c.provider.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required for local provider")
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)

		isOllamaAPI := c.isOllamaAPI()
		debugPrint("ProcessMessageStream: isOllamaAPI=%v, endpoint=%s\n", isOllamaAPI, c.provider.Endpoint)

		if !isOllamaAPI {
			// For non-Ollama local models, fall back to non-streaming
			result, err := c.ProcessMessage(message)
			if err != nil {
				ch <- StreamChunk{Error: err}
			} else {
				ch <- StreamChunk{Content: result}
				ch <- StreamChunk{Done: true}
			}
			return
		}

		// For Ollama, use streaming
		ollamaReq := OllamaRequest{
			Model:  c.provider.Model,
			Stream: true,
			Messages: []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				{Role: "user", Content: message},
			},
		}

		jsonData, err := json.Marshal(ollamaReq)
		if err != nil {
			ch <- StreamChunk{Error: fmt.Errorf("failed to marshal request: %v", err)}
			return
		}

		req, err := http.NewRequest("POST", c.provider.Endpoint, bytes.NewBuffer(jsonData))
		if err != nil {
			ch <- StreamChunk{Error: fmt.Errorf("failed to create request: %v", err)}
			return
		}

		req.Header.Set(HeaderContentType, ContentTypeJSON)
		if c.provider.APIKey != "" {
			req.Header.Set(HeaderAuthorization, AuthBearerPrefix+c.provider.APIKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			ch <- StreamChunk{Error: fmt.Errorf("failed to send request: %v", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != StatusOK {
			body, _ := io.ReadAll(resp.Body)
			ch <- StreamChunk{Error: fmt.Errorf(ErrAPIRequestFailed, resp.StatusCode, string(body))}
			return
		}

		// Read streaming response line by line
		reader := bufio.NewReader(resp.Body)
		debugPrint("Starting to read streaming response\n")

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					ch <- StreamChunk{Error: fmt.Errorf("error reading stream: %v", err)}
				}
				break
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			debugPrint("Received line: %s\n", line)

			var resp OllamaResponse
			if err := json.Unmarshal([]byte(line), &resp); err != nil {
				debugPrint("Failed to parse line as JSON: %v\n", err)
				continue // Skip malformed chunks
			}

			if resp.Message.Content != "" {
				debugPrint("Sending content chunk: %s\n", resp.Message.Content)
				ch <- StreamChunk{Content: resp.Message.Content}
			}

			if resp.Done {
				debugPrint("Received done signal\n")
				ch <- StreamChunk{Done: true}
				break
			}
		}
	}()

	return ch, nil
}

// ProcessConversationStream processes conversation and streams the response
func (c *LocalClient) ProcessConversationStream(session *types.ConversationSession) (<-chan StreamChunk, error) {
	if c.provider.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required for local provider")
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)

		isOllamaAPI := c.isOllamaAPI()
		if !isOllamaAPI {
			// For non-Ollama local models, fall back to non-streaming
			result, err := c.ProcessConversation(session)
			if err != nil {
				ch <- StreamChunk{Error: err}
			} else {
				ch <- StreamChunk{Content: result}
				ch <- StreamChunk{Done: true}
			}
			return
		}

		// Convert conversation to messages
		messages := c.convertConversationToLocal(session, 5)

		// For Ollama, use streaming
		ollamaReq := OllamaRequest{
			Model:  c.provider.Model,
			Stream: true,
		}

		for _, msg := range messages {
			ollamaReq.Messages = append(ollamaReq.Messages, struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}

		jsonData, err := json.Marshal(ollamaReq)
		if err != nil {
			ch <- StreamChunk{Error: fmt.Errorf("failed to marshal request: %v", err)}
			return
		}

		req, err := http.NewRequest("POST", c.provider.Endpoint, bytes.NewBuffer(jsonData))
		if err != nil {
			ch <- StreamChunk{Error: fmt.Errorf("failed to create request: %v", err)}
			return
		}

		req.Header.Set(HeaderContentType, ContentTypeJSON)
		if c.provider.APIKey != "" {
			req.Header.Set(HeaderAuthorization, AuthBearerPrefix+c.provider.APIKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			ch <- StreamChunk{Error: fmt.Errorf("failed to send request: %v", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != StatusOK {
			body, _ := io.ReadAll(resp.Body)
			ch <- StreamChunk{Error: fmt.Errorf(ErrAPIRequestFailed, resp.StatusCode, string(body))}
			return
		}

		// Read streaming response line by line
		reader := bufio.NewReader(resp.Body)
		debugPrint("Starting to read streaming response\n")

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					ch <- StreamChunk{Error: fmt.Errorf("error reading stream: %v", err)}
				}
				break
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			debugPrint("Received line: %s\n", line)

			var resp OllamaResponse
			if err := json.Unmarshal([]byte(line), &resp); err != nil {
				debugPrint("Failed to parse line as JSON: %v\n", err)
				continue // Skip malformed chunks
			}

			if resp.Message.Content != "" {
				debugPrint("Sending content chunk: %s\n", resp.Message.Content)
				ch <- StreamChunk{Content: resp.Message.Content}
			}

			if resp.Done {
				debugPrint("Received done signal\n")
				ch <- StreamChunk{Done: true}
				break
			}
		}
	}()

	return ch, nil
}

// SupportsStreaming indicates if this client supports streaming
func (c *LocalClient) SupportsStreaming() bool {
	return c.isOllamaAPI() // Only Ollama supports streaming for now
}

// getMaxTokensForLocalModel returns a conservative token limit for local models
func getMaxTokensForLocalModel(model string) int {
	// Most local models have smaller context windows
	return LocalModelDefaultTokens
}
