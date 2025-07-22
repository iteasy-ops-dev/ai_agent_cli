package llm

import (
	"fmt"
	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
	"github.com/iteasy-ops-dev/syseng-agent/pkg/utils"
)

type OpenAIRequest struct {
	Model      string      `json:"model"`
	Messages   []Message   `json:"messages"`
	Tools      []Tool      `json:"tools,omitempty"`
	ToolChoice interface{} `json:"tool_choice,omitempty"`
	Stream     bool        `json:"stream,omitempty"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"` // For tool messages
}

type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type OpenAIResponse struct {
	Choices []Choice  `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

type Choice struct {
	Message Message `json:"message"`
}

type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// OpenAIStreamResponse represents streaming response from OpenAI
type OpenAIStreamResponse struct {
	Choices []StreamChoice `json:"choices"`
}

type StreamChoice struct {
	Delta Delta `json:"delta"`
}

type Delta struct {
	Content string `json:"content"`
}

func ConvertMCPToolsToOpenAI(mcpTools []map[string]interface{}) []Tool {
	var tools []Tool

	for _, mcpTool := range mcpTools {
		tool := Tool{
			Type: "function",
			Function: ToolFunction{
				Name:        utils.GetString(mcpTool, "name"),
				Description: utils.GetString(mcpTool, "description"),
				Parameters:  utils.GetMap(mcpTool, "inputSchema"),
			},
		}
		tools = append(tools, tool)
	}

	return tools
}

// ConvertConversationToOpenAI converts a conversation session to OpenAI message format
func ConvertConversationToOpenAI(session *types.ConversationSession) []Message {
	return ConvertConversationToOpenAIWithTools(session, 0)
}

// ConvertConversationToOpenAIWithTools converts a conversation session to OpenAI message format with tool count info
func ConvertConversationToOpenAIWithTools(session *types.ConversationSession, toolCount int) []Message {
	var messages []Message
	
	// Add improved system prompt that encourages appropriate tool usage
	systemPrompt := fmt.Sprintf(`You are a system engineer AI assistant with access to powerful desktop tools.

CURRENT STATUS: You have %d tools available.

TOOL USAGE PRIORITY:
1. ALWAYS use tools for system operations, even if they seem simple:
   - Network operations (ping, curl, wget, etc.)
   - File operations (read, write, list, search, etc.)  
   - Process management (ps, kill, start, etc.)
   - System information (uname, df, free, etc.)
   - Command execution (any shell command)

2. Use conversation context for follow-up questions:
   - "What did I just do?" → Answer from conversation history
   - "Where did you save that file?" → Reference previous actions
   - "Show me that again" → Use previous results

3. Answer directly WITHOUT tools only for:
   - Pure greetings (안녕, hello, hi)
   - General knowledge questions unrelated to system
   - Questions about previous conversation content

CRITICAL EXAMPLES:
• "ping google" → USE start_process tool
• "check disk space" → USE start_process tool  
• "list files" → USE list_directory tool
• "what's running" → USE start_process tool
• "안녕" → Answer directly
• "방금 뭐했어?" → Answer from conversation history

ERROR HANDLING STRATEGY:
• When a tool call fails, ALWAYS try alternative approaches before giving up
• If a path doesn't exist, try common alternative locations
• If a command fails, try variations or related commands
• Provide helpful suggestions even when tools fail

COMMON FALLBACK PATHS:
• Downloads: Try in order: ~/Downloads, ~/다운로드, ~/Desktop, ~/Documents, /tmp, $HOME
• System info: Try multiple commands: uname, lsb_release, cat /etc/os-release
• Processes: Try ps, top, htop, systemctl commands as appropriate

Remember: Use tools proactively! Don't just describe what you COULD do - actually DO it with the tools available.`, toolCount)

	messages = append(messages, Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// Add conversation messages
	for _, msg := range session.Messages {
		messages = append(messages, Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return messages
}