package types

import (
	"time"
)

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"inputSchema"`
}

type MCPServer struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Transport   string            `json:"transport"`
	Status      string            `json:"status"`
	Capabilities []string         `json:"capabilities"`
	Metadata    map[string]string `json:"metadata"`
	Tools       []Tool            `json:"tools,omitempty"`
	LastPing    time.Time         `json:"last_ping"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type LLMProvider struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	APIKey    string            `json:"api_key"`
	Endpoint  string            `json:"endpoint"`
	Model     string            `json:"model"`
	Config    map[string]interface{} `json:"config"`
	IsActive  bool              `json:"is_active"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type AgentRequest struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Message     string                 `json:"message"`
	Context     map[string]interface{} `json:"context"`
	MCPServerID string                 `json:"mcp_server_id,omitempty"`
	ProviderID  string                 `json:"provider_id,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

type AgentResponse struct {
	ID        string                 `json:"id"`
	RequestID string                 `json:"request_id"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// ConversationMessage represents a single message in a conversation
type ConversationMessage struct {
	Role      string    `json:"role"`      // "user", "assistant", "system", "tool"
	Content   string    `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string   `json:"tool_call_id,omitempty"`
	Name      string    `json:"name,omitempty"` // For tool messages
	Timestamp time.Time `json:"timestamp"`
}

// ToolCall represents a function/tool call
type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function ToolCallFunction       `json:"function"`
}

// ToolCallFunction represents the function details in a tool call
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ConversationSession manages the state of a conversation
type ConversationSession struct {
	ID           string                `json:"id"`
	UserID       string                `json:"user_id,omitempty"`
	Messages     []ConversationMessage `json:"messages"`
	MCPServerID  string                `json:"mcp_server_id,omitempty"`
	ProviderID   string                `json:"provider_id,omitempty"`
	Interactive  bool                  `json:"interactive"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

// AddMessage adds a message to the conversation session
func (cs *ConversationSession) AddMessage(role, content string) {
	cs.Messages = append(cs.Messages, ConversationMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	cs.UpdatedAt = time.Now()
}

// AddToolCall adds a tool call message to the conversation
func (cs *ConversationSession) AddToolCall(toolCalls []ToolCall) {
	cs.Messages = append(cs.Messages, ConversationMessage{
		Role:      "assistant",
		Content:   "",
		ToolCalls: toolCalls,
		Timestamp: time.Now(),
	})
	cs.UpdatedAt = time.Now()
}

// AddToolResult adds a tool result message to the conversation
func (cs *ConversationSession) AddToolResult(toolCallID, toolName, result string) {
	cs.Messages = append(cs.Messages, ConversationMessage{
		Role:       "tool",
		Content:    result,
		ToolCallID: toolCallID,
		Name:       toolName,
		Timestamp:  time.Now(),
	})
	cs.UpdatedAt = time.Now()
}

// GetLastUserMessage returns the last user message content
func (cs *ConversationSession) GetLastUserMessage() string {
	for i := len(cs.Messages) - 1; i >= 0; i-- {
		if cs.Messages[i].Role == "user" {
			return cs.Messages[i].Content
		}
	}
	return ""
}

type Config struct {
	Server struct {
		Port string `mapstructure:"port"`
		Host string `mapstructure:"host"`
	} `mapstructure:"server"`
	
	Database struct {
		Type string `mapstructure:"type"`
		Path string `mapstructure:"path"`
	} `mapstructure:"database"`
	
	Logging struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"logging"`
	
	Agent struct {
		DefaultProvider string `mapstructure:"default_provider"`
		Timeout         int    `mapstructure:"timeout"`
	} `mapstructure:"agent"`
}