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