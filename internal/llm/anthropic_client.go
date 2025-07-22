package llm

import (
	"fmt"

	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
)

// AnthropicRequest represents the request structure for Anthropic API
type AnthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []AnthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	Stream    bool               `json:"stream,omitempty"`
}

// AnthropicMessage represents a message in Anthropic format
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse represents the response from Anthropic API
type AnthropicResponse struct {
	Content []AnthropicContent `json:"content"`
	Model   string             `json:"model"`
	Usage   map[string]int     `json:"usage,omitempty"`
}

// AnthropicContent represents content block in Anthropic response
type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicStreamEvent represents a streaming event from Anthropic API
type AnthropicStreamEvent struct {
	Type  string                `json:"type"`
	Index int                   `json:"index,omitempty"`
	Delta *AnthropicStreamDelta `json:"delta,omitempty"`
}

// AnthropicStreamDelta represents the delta content in streaming
type AnthropicStreamDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicClient implements LLMClient for Anthropic models
type AnthropicClient struct {
	provider   *types.LLMProvider
	httpClient *HTTPClient
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient(provider *types.LLMProvider) *AnthropicClient {
	headers := map[string]string{
		HeaderAPIKey:           provider.APIKey,
		HeaderAnthropicVersion: AnthropicAPIVersion,
	}

	httpClient := NewHTTPClient(HTTPConfig{
		Timeout: DefaultHTTPTimeout,
		Headers: headers,
	})

	return &AnthropicClient{
		provider:   provider,
		httpClient: httpClient,
	}
}

// ProcessMessage processes a simple message
func (c *AnthropicClient) ProcessMessage(message string) (string, error) {
	if c.provider.APIKey == "" {
		return "", fmt.Errorf(ErrAPIKeyRequired, ProviderAnthropic)
	}

	endpoint := AnthropicMessagesURL
	if c.provider.Endpoint != "" {
		endpoint = c.provider.Endpoint
	}

	reqBody := AnthropicRequest{
		Model:     c.provider.Model,
		MaxTokens: AnthropicDefaultMaxTokens,
		Messages: []AnthropicMessage{
			{Role: RoleUser, Content: message},
		},
	}

	return c.executeRequest(endpoint, reqBody)
}

// ProcessWithTools processes a message with tools (currently limited support)
func (c *AnthropicClient) ProcessWithTools(message string, tools []Tool, toolCaller ToolCaller) (string, error) {
	// Anthropic doesn't have native function calling like OpenAI yet
	// We'll enhance the message with tool information as context
	if len(tools) > 0 {
		toolContext := "Available tools:\n"
		for _, tool := range tools {
			toolContext += fmt.Sprintf("- %s: %s\n", tool.Function.Name, tool.Function.Description)
		}
		message = toolContext + "\n" + message + "\n\nPlease respond in a way that indicates if you would use any of these tools and how."
	}

	return c.ProcessMessage(message)
}

// ProcessConversation processes a message within conversation context
func (c *AnthropicClient) ProcessConversation(session *types.ConversationSession) (string, error) {
	if c.provider.APIKey == "" {
		return "", fmt.Errorf(ErrAPIKeyRequired, ProviderAnthropic)
	}

	endpoint := AnthropicMessagesURL
	if c.provider.Endpoint != "" {
		endpoint = c.provider.Endpoint
	}

	// Convert conversation to Anthropic format
	messages := c.convertConversationToAnthropic(session)

	reqBody := AnthropicRequest{
		Model:     c.provider.Model,
		MaxTokens: AnthropicDefaultMaxTokens,
		Messages:  messages,
		System:    DefaultSystemPrompt + " Maintain context from the conversation history.",
	}

	return c.executeRequest(endpoint, reqBody)
}

// ProcessConversationWithTools processes conversation with tools
func (c *AnthropicClient) ProcessConversationWithTools(session *types.ConversationSession, tools []Tool, toolCaller ToolCaller) (string, error) {
	// For now, add tool context to the system prompt
	systemPrompt := DefaultSystemPrompt + " Maintain context from the conversation history."

	if len(tools) > 0 {
		systemPrompt += "\n\nAvailable tools:\n"
		for _, tool := range tools {
			systemPrompt += fmt.Sprintf("- %s: %s\n", tool.Function.Name, tool.Function.Description)
		}
		systemPrompt += "\nWhen you need to use tools, clearly indicate which tool you would use and with what parameters."
	}

	if c.provider.APIKey == "" {
		return "", fmt.Errorf(ErrAPIKeyRequired, ProviderAnthropic)
	}

	endpoint := AnthropicMessagesURL
	if c.provider.Endpoint != "" {
		endpoint = c.provider.Endpoint
	}

	messages := c.convertConversationToAnthropic(session)

	reqBody := AnthropicRequest{
		Model:     c.provider.Model,
		MaxTokens: AnthropicDefaultMaxTokens,
		Messages:  messages,
		System:    systemPrompt,
	}

	return c.executeRequest(endpoint, reqBody)
}

// convertConversationToAnthropic converts conversation session to Anthropic format
func (c *AnthropicClient) convertConversationToAnthropic(session *types.ConversationSession) []AnthropicMessage {
	var messages []AnthropicMessage

	for _, msg := range session.Messages {
		// Anthropic uses "user" and "assistant" roles
		role := msg.Role
		if role == "system" {
			// Skip system messages as they go in the system field
			continue
		}

		messages = append(messages, AnthropicMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	return messages
}

// executeRequest executes a request to Anthropic API
func (c *AnthropicClient) executeRequest(endpoint string, reqBody AnthropicRequest) (string, error) {
	resp, err := c.httpClient.PostJSON(endpoint, reqBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var anthropicResp AnthropicResponse
	if err := UnmarshalJSONResponse(resp, &anthropicResp); err != nil {
		return "", err
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf(ErrNoContentInResponse)
	}

	// Return the first text content block
	for _, content := range anthropicResp.Content {
		if content.Type == ContentTypeText {
			return content.Text, nil
		}
	}

	return "", fmt.Errorf(ErrNoTextContentFound)
}

// Interface compliance methods

func (c *AnthropicClient) GetProviderInfo() ProviderInfo {
	return ProviderInfo{
		Type:     c.provider.Type,
		Model:    c.provider.Model,
		Endpoint: c.provider.Endpoint,
		Version:  AnthropicAPIVersion,
	}
}

func (c *AnthropicClient) IsHealthy() bool {
	return c.provider.APIKey != ""
}

func (c *AnthropicClient) SupportsFunctionCalling() bool {
	// Anthropic doesn't have native function calling yet
	return false
}

func (c *AnthropicClient) SupportsConversation() bool {
	return true
}

func (c *AnthropicClient) GetCapabilities() ClientCapabilities {
	return ClientCapabilities{
		SupportsTools:        false, // No native tool support yet
		SupportsConversation: true,
		SupportsStreaming:    true,
		MaxTokens:            getMaxTokensForAnthropicModel(c.provider.Model),
		MaxConversationTurn:  AnthropicMaxConversationTurns,
	}
}

// ProcessMessageStream processes a message and streams the response
func (c *AnthropicClient) ProcessMessageStream(message string) (<-chan StreamChunk, error) {
	if c.provider.APIKey == "" {
		return nil, fmt.Errorf(ErrAPIKeyRequired, ProviderAnthropic)
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)

		endpoint := AnthropicMessagesURL
		if c.provider.Endpoint != "" {
			endpoint = c.provider.Endpoint
		}

		reqBody := AnthropicRequest{
			Model:     c.provider.Model,
			MaxTokens: 4096,
			Messages: []AnthropicMessage{
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
		parser := &AnthropicSSEParser{}

		if err := streamHandler.ProcessSSEStream(ch, parser); err != nil {
			ch <- StreamChunk{Error: err}
		}
	}()

	return ch, nil
}

// ProcessConversationStream processes conversation and streams the response
func (c *AnthropicClient) ProcessConversationStream(session *types.ConversationSession) (<-chan StreamChunk, error) {
	if c.provider.APIKey == "" {
		return nil, fmt.Errorf(ErrAPIKeyRequired, ProviderAnthropic)
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)

		// Convert conversation to Anthropic format
		messages := c.convertConversationToAnthropic(session)
		systemPrompt := DefaultSystemPrompt + " Maintain context from the conversation history."

		endpoint := AnthropicMessagesURL
		if c.provider.Endpoint != "" {
			endpoint = c.provider.Endpoint
		}

		reqBody := AnthropicRequest{
			Model:     c.provider.Model,
			MaxTokens: 4096,
			Messages:  messages,
			System:    systemPrompt,
			Stream:    true,
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
		parser := &AnthropicSSEParser{}

		if err := streamHandler.ProcessSSEStream(ch, parser); err != nil {
			ch <- StreamChunk{Error: err}
		}
	}()

	return ch, nil
}

// SupportsStreaming indicates if this client supports streaming
func (c *AnthropicClient) SupportsStreaming() bool {
	return true
}

// getMaxTokensForAnthropicModel returns the maximum token limit for Anthropic models
func getMaxTokensForAnthropicModel(model string) int {
	switch model {
	case ModelClaude3Opus:
		return Claude3OpusTokens
	case ModelClaude3Sonnet:
		return Claude3SonnetTokens
	case ModelClaude3Haiku:
		return Claude3HaikuTokens
	case ModelClaude21:
		return Claude21Tokens
	case ModelClaude20:
		return Claude20Tokens
	case ModelClaudeInstant:
		return ClaudeInstantTokens
	default:
		return ClaudeInstantTokens
	}
}
