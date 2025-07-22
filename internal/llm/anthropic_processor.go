package llm

import (
	"fmt"
	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
	"github.com/iteasy-ops-dev/syseng-agent/internal/ui"
)

// AnthropicProcessor implements LLMProcessor for Anthropic models using the new client interface
type AnthropicProcessor struct {
	*BaseProcessor
	client LLMClient
}

// NewAnthropicProcessor creates a new AnthropicProcessor
func NewAnthropicProcessor(provider *types.LLMProvider, promptManager PromptManager) *AnthropicProcessor {
	client := NewAnthropicClient(provider)
	return &AnthropicProcessor{
		BaseProcessor: NewBaseProcessor(provider, promptManager),
		client:        client,
	}
}

// ProcessWithTools processes a message with tools using Anthropic
func (p *AnthropicProcessor) ProcessWithTools(message string, tools []Tool, toolCaller ToolCaller) (string, error) {
	enhancedMessage := p.buildEnhancedMessage(message, len(tools))
	
	// Use the new client interface
	if toolSupport, ok := p.client.(ToolSupport); ok {
		return toolSupport.ProcessWithTools(enhancedMessage, tools, toolCaller)
	}
	
	// Fallback to simple message processing
	return p.client.ProcessMessage(enhancedMessage)
}

// ProcessConversation processes a conversation with tools using Anthropic
func (p *AnthropicProcessor) ProcessConversation(session *types.ConversationSession, tools []Tool, toolCaller ToolCaller) (string, error) {
	// Use the new client interface
	if conversationSupport, ok := p.client.(ConversationSupport); ok {
		if len(tools) == 0 {
			return conversationSupport.ProcessConversation(session)
		}
		return conversationSupport.ProcessConversationWithTools(session, tools, toolCaller)
	}
	
	// Fallback for clients that don't support conversation
	lastMessage := session.GetLastUserMessage()
	return p.ProcessWithTools(lastMessage, tools, toolCaller)
}

// ProcessWithUI processes a message with tools and UI feedback
func (p *AnthropicProcessor) ProcessWithUI(message string, tools []Tool, toolCaller ToolCaller, display ui.ToolDisplayInterface) (string, error) {
	return p.processWithToolsCommon(p.client, message, tools, toolCaller, display)
}

// ProcessConversationWithUI processes a conversation with tools and UI feedback
func (p *AnthropicProcessor) ProcessConversationWithUI(session *types.ConversationSession, tools []Tool, toolCaller ToolCaller, display ui.ToolDisplayInterface) (string, error) {
	return p.processConversationCommon(p.client, session, tools, toolCaller, display)
}

// buildConversationContext creates a conversation context string for Anthropic
func (p *AnthropicProcessor) buildConversationContext(session *types.ConversationSession) string {
	if len(session.Messages) <= 1 {
		return ""
	}
	
	// Build a simple conversation history context
	contextPrompt := p.getConversationPrompt()
	
	// Add recent conversation history (last few messages)
	messageCount := len(session.Messages)
	startIdx := messageCount - 3
	if startIdx < 0 {
		startIdx = 0
	}
	
	if messageCount > 1 {
		contextPrompt += "\n\nRecent conversation context:"
		for i := startIdx; i < messageCount; i++ {
			msg := session.Messages[i]
			contextPrompt += fmt.Sprintf("\n%s: %s", msg.Role, msg.Content)
		}
	}
	
	return contextPrompt
}


// Capability methods
func (p *AnthropicProcessor) SupportsConversation() bool {
	return true // Basic conversation support through context management
}

func (p *AnthropicProcessor) SupportsFunctionCalling() bool {
	return true // Through unified CallAnthropicWithTools
}

func (p *AnthropicProcessor) SupportsUIFeedback() bool {
	return true
}

// GetClient returns the underlying LLM client
func (p *AnthropicProcessor) GetClient() LLMClient {
	return p.client
}