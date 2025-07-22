package llm

import (
	"fmt"
	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
	"github.com/iteasy-ops-dev/syseng-agent/internal/ui"
)

// LocalProcessor implements LLMProcessor for local LLM models using the new client interface
type LocalProcessor struct {
	*BaseProcessor
	client LLMClient
}

// NewLocalProcessor creates a new LocalProcessor
func NewLocalProcessor(provider *types.LLMProvider, promptManager PromptManager) *LocalProcessor {
	client := NewLocalClient(provider)
	return &LocalProcessor{
		BaseProcessor: NewBaseProcessor(provider, promptManager),
		client:        client,
	}
}

// ProcessWithTools processes a message with tools using Local LLM
func (p *LocalProcessor) ProcessWithTools(message string, tools []Tool, toolCaller ToolCaller) (string, error) {
	enhancedMessage := p.buildEnhancedMessage(message, len(tools))
	
	// Use the new client interface
	if toolSupport, ok := p.client.(ToolSupport); ok {
		return toolSupport.ProcessWithTools(enhancedMessage, tools, toolCaller)
	}
	
	// Fallback to simple message processing
	return p.client.ProcessMessage(enhancedMessage)
}

// ProcessConversation processes a conversation with tools using Local LLM
func (p *LocalProcessor) ProcessConversation(session *types.ConversationSession, tools []Tool, toolCaller ToolCaller) (string, error) {
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
func (p *LocalProcessor) ProcessWithUI(message string, tools []Tool, toolCaller ToolCaller, display ui.ToolDisplayInterface) (string, error) {
	return p.processWithToolsCommon(p.client, message, tools, toolCaller, display)
}

// ProcessConversationWithUI processes a conversation with tools and UI feedback
func (p *LocalProcessor) ProcessConversationWithUI(session *types.ConversationSession, tools []Tool, toolCaller ToolCaller, display ui.ToolDisplayInterface) (string, error) {
	return p.processConversationCommon(p.client, session, tools, toolCaller, display)
}

// buildLocalConversationContext creates a minimal conversation context for local models
func (p *LocalProcessor) buildLocalConversationContext(session *types.ConversationSession) string {
	if len(session.Messages) <= 1 {
		return ""
	}
	
	// Keep it simple for local models - just include the basic conversation prompt
	conversationPrompt := p.getConversationPrompt()
	
	// For local models, only include the most recent exchange to avoid overwhelming the context
	messageCount := len(session.Messages)
	startIdx := messageCount - 2 // Get last 2 messages only
	if startIdx < 0 {
		startIdx = 0
	}
	
	if messageCount > 1 {
		conversationPrompt += "\n\nPrevious exchange:"
		for i := startIdx; i < messageCount; i++ {
			msg := session.Messages[i]
			// Keep it concise for local models
			truncatedContent := msg.Content
			if len(truncatedContent) > LocalContentTruncateLimit {
				truncatedContent = truncatedContent[:LocalContentTruncateLimit] + "..."
			}
			conversationPrompt += fmt.Sprintf("\n%s: %s", msg.Role, truncatedContent)
		}
	}
	
	return conversationPrompt
}


// Capability methods
func (p *LocalProcessor) SupportsConversation() bool {
	return true // Basic conversation support with simplified context
}

func (p *LocalProcessor) SupportsFunctionCalling() bool {
	return true // Through OpenAI-compatible interface
}

func (p *LocalProcessor) SupportsUIFeedback() bool {
	return true
}

// GetClient returns the underlying LLM client
func (p *LocalProcessor) GetClient() LLMClient {
	return p.client
}