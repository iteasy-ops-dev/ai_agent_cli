package llm

import (
	"fmt"
	"strings"
)

// PromptManager manages LLM-specific prompts and templates
type PromptManager interface {
	GetSystemPrompt(llmType string) string
	GetToolContextPrompt(toolCount int) string
	GetErrorHintPrompt(errorType string) string
	GetConversationPrompt(llmType string) string
	BuildEnhancedMessage(baseMessage, context string) string
}

// PromptTemplate contains templates for different types of prompts
type PromptTemplate struct {
	System       string
	ToolContext  string
	ErrorHints   map[string]string
	Conversation string
}

// DefaultPromptManager implements PromptManager with predefined templates
type DefaultPromptManager struct {
	templates map[string]*PromptTemplate
}

// NewDefaultPromptManager creates a new DefaultPromptManager with predefined templates
func NewDefaultPromptManager() *DefaultPromptManager {
	pm := &DefaultPromptManager{
		templates: make(map[string]*PromptTemplate),
	}
	pm.initializeTemplates()
	return pm
}

// initializeTemplates sets up the predefined prompt templates for each LLM type
func (pm *DefaultPromptManager) initializeTemplates() {
	// OpenAI templates
	pm.templates["openai"] = &PromptTemplate{
		System: `You are a helpful AI assistant with access to system tools. 
Use the available tools to provide comprehensive and accurate responses. 
When using tools, explain what you're doing and why.`,
		ToolContext: `Available tools: %d desktop tools including file operations, system commands, and process management.
Use multiple tools as needed to provide comprehensive answers.`,
		ErrorHints: map[string]string{
			"file_not_found":    "The file or directory doesn't exist. Try checking the correct path or suggest alternatives.",
			"permission_denied": "Permission denied. Consider suggesting alternative approaches or checking file permissions.",
			"network_error":     "Network connectivity issue. Try alternative approaches or suggest checking network connection.",
			"tool_error":        "Tool execution failed. Try alternative tools or approaches to accomplish the task.",
		},
		Conversation: `Continue the conversation naturally, maintaining context from previous messages.
Use available tools when needed to provide accurate and helpful responses.`,
	}

	// Anthropic templates
	pm.templates["anthropic"] = &PromptTemplate{
		System: `You are Claude, a helpful AI assistant. You have access to various system tools
that allow you to interact with the desktop environment. Use these tools thoughtfully
to provide accurate and comprehensive responses.`,
		ToolContext: `You have access to %d system tools for file operations, process management, and system commands.
Utilize these tools effectively to gather information and complete tasks.`,
		ErrorHints: map[string]string{
			"file_not_found":    "The requested file or directory could not be found. Please verify the path is correct.",
			"permission_denied": "Access denied to the requested resource. Consider alternative approaches.",
			"network_error":     "Network connection issue encountered. Consider offline alternatives if available.",
			"tool_error":        "Tool execution encountered an error. Consider using alternative methods.",
		},
		Conversation: `Maintain conversation context and provide helpful, accurate responses.
Leverage available tools to enhance your responses with real-time information.`,
	}

	// Local LLM templates (simplified for better performance)
	pm.templates["local"] = &PromptTemplate{
		System: `You are a helpful AI assistant with access to system tools. 
Use the available tools to provide comprehensive and accurate responses. 
When using tools, explain what you're doing and why.`,
		ToolContext: `Available tools: %d desktop tools including file operations, system commands, and process management.
Use multiple tools as needed to provide comprehensive answers.`,
		ErrorHints: map[string]string{
			"file_not_found":    "Try list_directory with different path",
			"permission_denied": "Try alternative approach or tool",
			"network_error":     "Use local tools only",
			"tool_error":        "Tool execution failed. Try alternative tools or approaches to accomplish the task.",
		},
		Conversation: `Continue the conversation naturally, maintaining context from previous messages.
Use available tools when needed to provide accurate and helpful responses.`,
	}

	// Default template for unknown LLM types
	pm.templates["default"] = pm.templates["openai"]
}

// GetSystemPrompt returns the system prompt for the specified LLM type
func (pm *DefaultPromptManager) GetSystemPrompt(llmType string) string {
	template := pm.getTemplate(llmType)
	return template.System
}

// GetToolContextPrompt returns the tool context prompt with the specified tool count
func (pm *DefaultPromptManager) GetToolContextPrompt(toolCount int) string {
	// Use OpenAI template as default for tool context
	template := pm.templates["openai"]
	return fmt.Sprintf(template.ToolContext, toolCount)
}

// GetErrorHintPrompt returns an appropriate error hint for the given error type
func (pm *DefaultPromptManager) GetErrorHintPrompt(errorType string) string {
	template := pm.templates["openai"] // Use OpenAI as default
	if hint, exists := template.ErrorHints[errorType]; exists {
		return hint
	}
	return template.ErrorHints["tool_error"] // Default error hint
}

// GetConversationPrompt returns the conversation prompt for the specified LLM type
func (pm *DefaultPromptManager) GetConversationPrompt(llmType string) string {
	template := pm.getTemplate(llmType)
	return template.Conversation
}

// BuildEnhancedMessage combines base message with context to create an enhanced prompt
func (pm *DefaultPromptManager) BuildEnhancedMessage(baseMessage, context string) string {
	if context == "" {
		return baseMessage
	}

	var builder strings.Builder
	builder.WriteString(baseMessage)
	builder.WriteString("\n\n")
	builder.WriteString(context)

	return builder.String()
}

// getTemplate returns the template for the specified LLM type, falling back to default
func (pm *DefaultPromptManager) getTemplate(llmType string) *PromptTemplate {
	if template, exists := pm.templates[strings.ToLower(llmType)]; exists {
		return template
	}
	return pm.templates["default"]
}
