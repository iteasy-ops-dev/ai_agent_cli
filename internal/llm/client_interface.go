package llm

import (
	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
)

// LLMClient defines the core interface that all LLM providers must implement
type LLMClient interface {
	// ProcessMessage processes a simple message and returns the response
	ProcessMessage(message string) (string, error)
	
	// GetProviderInfo returns information about the LLM provider
	GetProviderInfo() ProviderInfo
	
	// IsHealthy checks if the client is ready to process requests
	IsHealthy() bool
}

// ToolSupport defines interface for LLM clients that support tool/function calling
type ToolSupport interface {
	// ProcessWithTools processes a message with available tools
	ProcessWithTools(message string, tools []Tool, toolCaller ToolCaller) (string, error)
	
	// SupportsFunctionCalling indicates if this client supports native function calling
	SupportsFunctionCalling() bool
}

// ConversationSupport defines interface for LLM clients that support conversation context
type ConversationSupport interface {
	// ProcessConversation processes a message within conversation context
	ProcessConversation(session *types.ConversationSession) (string, error)
	
	// ProcessConversationWithTools processes conversation with tools
	ProcessConversationWithTools(session *types.ConversationSession, tools []Tool, toolCaller ToolCaller) (string, error)
	
	// SupportsConversation indicates if this client supports conversation context
	SupportsConversation() bool
}

// StreamingSupport defines interface for LLM clients that support streaming responses
type StreamingSupport interface {
	// ProcessMessageStream processes a message and streams the response
	ProcessMessageStream(message string) (<-chan StreamChunk, error)
	
	// ProcessConversationStream processes conversation and streams the response
	ProcessConversationStream(session *types.ConversationSession) (<-chan StreamChunk, error)
	
	// SupportsStreaming indicates if this client supports streaming
	SupportsStreaming() bool
}

// StreamChunk represents a chunk of streaming response
type StreamChunk struct {
	Content string
	Error   error
	Done    bool
}

// ProviderInfo contains information about the LLM provider
type ProviderInfo struct {
	Type     string
	Model    string
	Endpoint string
	Version  string
}

// ClientCapabilities aggregates all supported capabilities
type ClientCapabilities struct {
	SupportsTools        bool
	SupportsConversation bool
	SupportsStreaming    bool
	MaxTokens           int
	MaxConversationTurn int
}

// FullClient combines all interfaces for clients that support everything
type FullClient interface {
	LLMClient
	ToolSupport
	ConversationSupport
	
	// GetCapabilities returns what this client can do
	GetCapabilities() ClientCapabilities
}

// ClientFactory creates LLM clients based on provider configuration
type ClientFactory interface {
	// CreateClient creates a client for the given provider
	CreateClient(provider *types.LLMProvider) (LLMClient, error)
	
	// SupportedTypes returns list of supported provider types
	SupportedTypes() []string
}