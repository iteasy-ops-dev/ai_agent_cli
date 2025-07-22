package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
)

// OpenAIClient implements FullClient for OpenAI API
type OpenAIClient struct {
	provider   *types.LLMProvider
	httpClient *HTTPClient
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(provider *types.LLMProvider) *OpenAIClient {
	headers := map[string]string{
		HeaderAuthorization: AuthBearerPrefix + provider.APIKey,
	}

	httpClient := NewHTTPClient(HTTPConfig{
		Timeout: DefaultHTTPTimeout,
		Headers: headers,
	})

	return &OpenAIClient{
		provider:   provider,
		httpClient: httpClient,
	}
}

// ProcessMessage processes a simple message
func (c *OpenAIClient) ProcessMessage(message string) (string, error) {
	if c.provider.APIKey == "" {
		return "", fmt.Errorf(ErrAPIKeyRequired, ProviderOpenAI)
	}

	endpoint := OpenAIChatCompletionsURL
	if c.provider.Endpoint != "" {
		endpoint = c.provider.Endpoint
	}

	reqBody := OpenAIRequest{
		Model: c.provider.Model,
		Messages: []Message{
			{Role: RoleUser, Content: message},
		},
	}

	resp, err := c.httpClient.PostJSON(endpoint, reqBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var openAIResp OpenAIResponse
	if err := UnmarshalJSONResponse(resp, &openAIResp); err != nil {
		return "", err
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf(ErrNoResponseChoices)
	}

	return openAIResp.Choices[0].Message.Content, nil
}

// ProcessWithTools processes a message with available tools
func (c *OpenAIClient) ProcessWithTools(message string, tools []Tool, toolCaller ToolCaller) (string, error) {
	if c.provider.APIKey == "" {
		return "", fmt.Errorf(ErrAPIKeyRequired, ProviderOpenAI)
	}

	endpoint := OpenAIChatCompletionsURL
	if c.provider.Endpoint != "" {
		endpoint = c.provider.Endpoint
	}

	// Start with enhanced user message
	messages := []Message{
		{Role: "user", Content: EnhanceMessageWithTools(message, len(tools))},
	}

	// Create tool processor for validation and execution
	toolProcessor := NewToolProcessor(tools, toolCaller)

	// Iterative conversation with tools (up to max iterations to prevent infinite loops)
	for iteration := 0; iteration < MaxToolIterations; iteration++ {
		reqBody := OpenAIRequest{
			Model:    c.provider.Model,
			Messages: messages,
		}

		// Add tools if available
		if len(tools) > 0 {
			reqBody.Tools = tools
			reqBody.ToolChoice = ToolChoiceAuto
		}

		resp, err := c.httpClient.PostJSON(endpoint, reqBody)
		if err != nil {
			return "", err
		}

		var openAIResp OpenAIResponse
		if err := UnmarshalJSONResponse(resp, &openAIResp); err != nil {
			return "", err
		}
		resp.Body.Close()

		if len(openAIResp.Choices) == 0 {
			return "", fmt.Errorf("no response choices received")
		}

		choice := openAIResp.Choices[0]
		messages = append(messages, choice.Message)

		// If no tool calls, we're done
		if len(choice.Message.ToolCalls) == 0 {
			return choice.Message.Content, nil
		}

		// Execute tool calls using utility functions
		for _, toolCall := range choice.Message.ToolCalls {
			if toolCaller == nil {
				continue
			}

			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				messages = append(messages, Message{
					Role:       RoleTool,
					Content:    fmt.Sprintf(ErrParsingArguments, err),
					ToolCallID: toolCall.ID,
					Name:       toolCall.Function.Name,
				})
				continue
			}

			result, err := toolProcessor.ExecuteTool(toolCall.Function.Name, args)
			if err != nil {
				messages = append(messages, Message{
					Role:       RoleTool,
					Content:    fmt.Sprintf("Error: %v", err),
					ToolCallID: toolCall.ID,
					Name:       toolCall.Function.Name,
				})
				continue
			}

			// Use utility function to format result
			resultStr := FormatToolResult(result)

			messages = append(messages, Message{
				Role:       RoleTool,
				Content:    resultStr,
				ToolCallID: toolCall.ID,
				Name:       toolCall.Function.Name,
			})
		}
	}

	return "", fmt.Errorf(ErrMaxIterationsExceeded, MaxToolIterations)
}

// ProcessConversation processes a message within conversation context
func (c *OpenAIClient) ProcessConversation(session *types.ConversationSession) (string, error) {
	messages := ConvertConversationToOpenAI(session)

	reqBody := OpenAIRequest{
		Model:    c.provider.Model,
		Messages: messages,
	}

	return c.executeRequest(reqBody)
}

// ProcessConversationWithTools processes conversation with tools
func (c *OpenAIClient) ProcessConversationWithTools(session *types.ConversationSession, tools []Tool, toolCaller ToolCaller) (string, error) {
	messages := ConvertConversationToOpenAIWithTools(session, len(tools))

	// Use the same tool processing logic as ProcessWithTools but with conversation messages
	return c.processToolConversation(messages, tools, toolCaller)
}

// processToolConversation handles the tool calling loop for both simple and conversation messages
func (c *OpenAIClient) processToolConversation(messages []Message, tools []Tool, toolCaller ToolCaller) (string, error) {
	if c.provider.APIKey == "" {
		return "", fmt.Errorf(ErrAPIKeyRequired, ProviderOpenAI)
	}

	endpoint := OpenAIChatCompletionsURL
	if c.provider.Endpoint != "" {
		endpoint = c.provider.Endpoint
	}

	// Create tool processor for validation and execution
	toolProcessor := NewToolProcessor(tools, toolCaller)

	// Iterative conversation with tools
	for iteration := 0; iteration < MaxToolIterations; iteration++ {
		reqBody := OpenAIRequest{
			Model:    c.provider.Model,
			Messages: messages,
		}

		if len(tools) > 0 {
			reqBody.Tools = tools
			reqBody.ToolChoice = ToolChoiceAuto
		}

		resp, err := c.httpClient.PostJSON(endpoint, reqBody)
		if err != nil {
			return "", err
		}

		var openAIResp OpenAIResponse
		if err := UnmarshalJSONResponse(resp, &openAIResp); err != nil {
			return "", err
		}
		resp.Body.Close()

		if len(openAIResp.Choices) == 0 {
			return "", fmt.Errorf("no response choices received")
		}

		choice := openAIResp.Choices[0]
		messages = append(messages, choice.Message)

		// If no tool calls, we're done
		if len(choice.Message.ToolCalls) == 0 {
			return choice.Message.Content, nil
		}

		// Execute tool calls using utility functions
		for _, toolCall := range choice.Message.ToolCalls {
			if toolCaller == nil {
				continue
			}

			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				messages = append(messages, Message{
					Role:       RoleTool,
					Content:    fmt.Sprintf(ErrParsingArguments, err),
					ToolCallID: toolCall.ID,
					Name:       toolCall.Function.Name,
				})
				continue
			}

			result, err := toolProcessor.ExecuteTool(toolCall.Function.Name, args)
			if err != nil {
				messages = append(messages, Message{
					Role:       RoleTool,
					Content:    fmt.Sprintf("Error: %v", err),
					ToolCallID: toolCall.ID,
					Name:       toolCall.Function.Name,
				})
				continue
			}

			// Use utility function to format result
			resultStr := FormatToolResult(result)

			messages = append(messages, Message{
				Role:       RoleTool,
				Content:    resultStr,
				ToolCallID: toolCall.ID,
				Name:       toolCall.Function.Name,
			})
		}
	}

	return "", fmt.Errorf(ErrMaxIterationsExceeded, MaxToolIterations)
}

// executeRequest executes a simple request without tools
func (c *OpenAIClient) executeRequest(reqBody OpenAIRequest) (string, error) {
	if c.provider.APIKey == "" {
		return "", fmt.Errorf(ErrAPIKeyRequired, ProviderOpenAI)
	}

	endpoint := OpenAIChatCompletionsURL
	if c.provider.Endpoint != "" {
		endpoint = c.provider.Endpoint
	}

	resp, err := c.httpClient.PostJSON(endpoint, reqBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var openAIResp OpenAIResponse
	if err := UnmarshalJSONResponse(resp, &openAIResp); err != nil {
		return "", err
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf(ErrNoResponseChoices)
	}

	return openAIResp.Choices[0].Message.Content, nil
}

// Interface compliance methods

func (c *OpenAIClient) GetProviderInfo() ProviderInfo {
	return ProviderInfo{
		Type:     c.provider.Type,
		Model:    c.provider.Model,
		Endpoint: c.provider.Endpoint,
		Version:  OpenAIAPIVersion,
	}
}

func (c *OpenAIClient) IsHealthy() bool {
	return c.provider.APIKey != ""
}

func (c *OpenAIClient) SupportsFunctionCalling() bool {
	return true
}

func (c *OpenAIClient) SupportsConversation() bool {
	return true
}

func (c *OpenAIClient) GetCapabilities() ClientCapabilities {
	return ClientCapabilities{
		SupportsTools:        true,
		SupportsConversation: true,
		SupportsStreaming:    true,
		MaxTokens:            getMaxTokensForModel(c.provider.Model),
		MaxConversationTurn:  MaxConversationTurns,
	}
}

// ProcessMessageStream processes a message and streams the response
func (c *OpenAIClient) ProcessMessageStream(message string) (<-chan StreamChunk, error) {
	if c.provider.APIKey == "" {
		return nil, fmt.Errorf(ErrAPIKeyRequired, ProviderOpenAI)
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)

		endpoint := OpenAIChatCompletionsURL
		if c.provider.Endpoint != "" {
			endpoint = c.provider.Endpoint
		}

		reqBody := OpenAIRequest{
			Model: c.provider.Model,
			Messages: []Message{
				{Role: RoleUser, Content: message},
			},
			Stream: true,
		}

		resp, err := c.httpClient.PostJSON(endpoint, reqBody)
		if err != nil {
			ch <- StreamChunk{Error: err}
			return
		}
		defer resp.Body.Close()

		if err := CheckStatusCode(resp, ""); err != nil {
			ch <- StreamChunk{Error: err}
			return
		}

		// Use streaming utilities
		streamHandler := NewStreamHandler(resp.Body)
		parser := &OpenAISSEParser{}

		if err := streamHandler.ProcessSSEStream(ch, parser); err != nil {
			ch <- StreamChunk{Error: err}
		}
	}()

	return ch, nil
}

// ProcessConversationStream processes conversation and streams the response
func (c *OpenAIClient) ProcessConversationStream(session *types.ConversationSession) (<-chan StreamChunk, error) {
	if c.provider.APIKey == "" {
		return nil, fmt.Errorf(ErrAPIKeyRequired, ProviderOpenAI)
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)

		messages := ConvertConversationToOpenAI(session)

		endpoint := OpenAIChatCompletionsURL
		if c.provider.Endpoint != "" {
			endpoint = c.provider.Endpoint
		}

		reqBody := OpenAIRequest{
			Model:    c.provider.Model,
			Messages: messages,
			Stream:   true,
		}

		resp, err := c.httpClient.PostJSON(endpoint, reqBody)
		if err != nil {
			ch <- StreamChunk{Error: err}
			return
		}
		defer resp.Body.Close()

		if err := CheckStatusCode(resp, ""); err != nil {
			ch <- StreamChunk{Error: err}
			return
		}

		// Use streaming utilities
		streamHandler := NewStreamHandler(resp.Body)
		parser := &OpenAISSEParser{}

		if err := streamHandler.ProcessSSEStream(ch, parser); err != nil {
			ch <- StreamChunk{Error: err}
		}
	}()

	return ch, nil
}

// SupportsStreaming indicates if this client supports streaming
func (c *OpenAIClient) SupportsStreaming() bool {
	return true
}

// getMaxTokensForModel returns the maximum token limit for OpenAI models
func getMaxTokensForModel(model string) int {
	switch {
	case strings.Contains(model, ModelGPT4):
		return GPT4TokenLimit
	case strings.Contains(model, ModelGPT35Turbo):
		return GPT35TurboTokenLimit
	default:
		return DefaultMaxTokens
	}
}
