package llm

import "time"

// API Endpoints
const (
	// OpenAI API endpoints
	OpenAIAPIBaseURL         = "https://api.openai.com/v1"
	OpenAIChatCompletionsURL = "https://api.openai.com/v1/chat/completions"

	// Anthropic API endpoints
	AnthropicAPIBaseURL  = "https://api.anthropic.com/v1"
	AnthropicMessagesURL = "https://api.anthropic.com/v1/messages"

	// API endpoint paths
	ChatCompletionsPath = "/chat/completions"
	MessagesPath        = "/messages"
	APIChatPath         = "/api/chat"
)

// HTTP Headers
const (
	HeaderAuthorization    = "Authorization"
	HeaderContentType      = "Content-Type"
	HeaderAPIKey           = "x-api-key"
	HeaderAnthropicVersion = "anthropic-version"

	// Content types
	ContentTypeJSON = "application/json"
	ContentTypeText = "text"
	ContentTypeSSE  = "text/event-stream"

	// Authorization prefixes
	AuthBearerPrefix = "Bearer "
)

// API Versions
const (
	AnthropicAPIVersion = "2023-06-01"
	OpenAIAPIVersion    = "v1"
)

// Timeout and Duration Values
const (
	DefaultHTTPTimeout   = 60 * time.Second
	LocalModelTimeout    = 120 * time.Second
	DefaultStreamTimeout = 300 * time.Second

	// Health check timeouts
	StdioHealthTimeout      = 5 * time.Minute
	PersistentHealthTimeout = 60 * time.Second
)

// Token Limits and Model Configurations
const (
	// Default token limits
	DefaultMaxTokens          = 4096
	AnthropicDefaultMaxTokens = 4096
	LocalModelDefaultTokens   = 4096

	// OpenAI model token limits
	GPT4TokenLimit       = 8192
	GPT35TurboTokenLimit = 4096

	// Anthropic model token limits
	Claude3OpusTokens   = 200000
	Claude3SonnetTokens = 200000
	Claude3HaikuTokens  = 200000
	Claude21Tokens      = 200000
	Claude20Tokens      = 100000
	ClaudeInstantTokens = 100000

	// Conversation limits
	MaxConversationTurns          = 50
	LocalMaxConversationTurns     = 20
	AnthropicMaxConversationTurns = 100

	// Iteration limits
	MaxToolIterations      = 10
	LocalMaxToolIterations = 10
)

// Model Names
const (
	// OpenAI Models
	ModelGPT4       = "gpt-4"
	ModelGPT35Turbo = "gpt-3.5-turbo"

	// Anthropic Models
	ModelClaude3Opus   = "claude-3-opus-20240229"
	ModelClaude3Sonnet = "claude-3-sonnet-20240229"
	ModelClaude3Haiku  = "claude-3-haiku-20240307"
	ModelClaude21      = "claude-2.1"
	ModelClaude20      = "claude-2.0"
	ModelClaudeInstant = "claude-instant-1.2"
)

// Provider Types
const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderLocal     = "local"
)

// Role Names
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleTool      = "tool"
)

// Tool Choice Options
const (
	ToolChoiceAuto = "auto"
	ToolChoiceNone = "none"
)

// Stream Event Types
const (
	// OpenAI streaming events
	StreamEventDelta = "delta"
	StreamEventDone  = "done"

	// Anthropic streaming events
	AnthropicEventContentBlock = "content_block_delta"
	AnthropicEventMessage      = "message_delta"
)

// Default Values
const (
	DefaultSystemPrompt        = "You are a helpful AI assistant."
	LocalContextWindowSize     = 8   // messages (increased from 5)
	AnthropicContextWindowSize = 5   // messages (increased from 3)
	LocalContentTruncateLimit  = 1000 // characters (increased from 200)
)

// Error Messages
const (
	ErrAPIKeyRequired         = "API key is required for %s"
	ErrEndpointRequired       = "endpoint is required for local provider"
	ErrNoResponseChoices      = "no response choices received"
	ErrNoContentInResponse    = "no content in response"
	ErrNoTextContentFound     = "no text content found in response"
	ErrMaxIterationsExceeded  = "conversation exceeded maximum iterations (%d)"
	ErrToolNotFound           = "Tool '%s' not found. Available tools: %s. Please use EXACT tool names from the list."
	ErrToolExecutionFailed    = "Tool execution failed: %v. Available tools: %s. Please use correct tool names and try again."
	ErrParsingArguments       = "Error parsing arguments: %v"
	ErrFailedToMarshal        = "failed to marshal request: %v"
	ErrFailedToCreateRequest  = "failed to create request: %v"
	ErrFailedToSendRequest    = "failed to send request: %v"
	ErrFailedToReadResponse   = "failed to read response body: %v"
	ErrAPIRequestFailed       = "API request failed with status %d: %s"
	ErrFailedToDecodeResponse = "failed to decode response: %v"
	ErrFailedToParseResponse  = "failed to parse response: %v"
	ErrStreamingReadError     = "error reading stream: %v"
	ErrUnableToParseResponse  = "unable to parse response body as any known format: %s"
	ErrNoContentInStream      = "no content found in streaming response"
)

// Debug and Environment Variables
const (
	EnvDebug   = "DEBUG"
	EnvVerbose = "VERBOSE"
)

// HTTP Status Codes (commonly used)
const (
	StatusOK = 200
)

// Ollama Detection Patterns
const (
	OllamaPortPattern = ":11434"
	OllamaNamePattern = "ollama"
	OllamaAPIPath     = "/api/chat"
)

// Tool Usage Patterns for Local Models
const (
	LocalToolPattern       = `{"use_tool":`
	LocalToolFormatExample = `{"use_tool": "EXACT_TOOL_NAME", "parameters": {"key": "value"}}`
)

// Progress Messages
const (
	ProgressInitializingAgent               = "Initializing AI agent..."
	ProgressFindingLLMProvider              = "Finding LLM provider..."
	ProgressProcessingWithLLM               = "Processing with %s (no tools available)..."
	ProgressProcessingWithTools             = "Processing with %s and available tools..."
	ProgressProcessingConversation          = "Processing conversation with %s (no tools available)..."
	ProgressProcessingConversationWithTools = "Processing conversation with %s and available tools..."
)

// Stream Response Markers
const (
	StreamDataPrefix = "data: "
	StreamDoneMarker = "[DONE]"
)

// JSON Field Names
const (
	JSONFieldContent    = "content"
	JSONFieldText       = "text"
	JSONFieldResponse   = "response"
	JSONFieldMessage    = "message"
	JSONFieldOutput     = "output"
	JSONFieldUseTool    = "use_tool"
	JSONFieldParameters = "parameters"
)
