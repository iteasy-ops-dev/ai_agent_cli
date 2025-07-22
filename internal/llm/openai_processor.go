package llm

import (
	"github.com/iteasy-ops-dev/syseng-agent/internal/ui"
	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
)

// OpenAIProcessor implements LLMProcessor for OpenAI models using the new client interface
type OpenAIProcessor struct {
	*BaseProcessor
	client LLMClient
}

// NewOpenAIProcessor creates a new OpenAIProcessor
func NewOpenAIProcessor(provider *types.LLMProvider, promptManager PromptManager) *OpenAIProcessor {
	client := NewOpenAIClient(provider)
	return &OpenAIProcessor{
		BaseProcessor: NewBaseProcessor(provider, promptManager),
		client:        client,
	}
}

// ProcessWithTools processes a message with tools using OpenAI
func (p *OpenAIProcessor) ProcessWithTools(message string, tools []Tool, toolCaller ToolCaller) (string, error) {
	enhancedMessage := p.buildEnhancedMessage(message, len(tools))

	// Use the new client interface
	if toolSupport, ok := p.client.(ToolSupport); ok {
		return toolSupport.ProcessWithTools(enhancedMessage, tools, toolCaller)
	}

	// Fallback to simple message processing
	return p.client.ProcessMessage(enhancedMessage)
}

// ProcessConversation processes a conversation with tools using OpenAI
func (p *OpenAIProcessor) ProcessConversation(session *types.ConversationSession, tools []Tool, toolCaller ToolCaller) (string, error) {
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
func (p *OpenAIProcessor) ProcessWithUI(message string, tools []Tool, toolCaller ToolCaller, display ui.ToolDisplayInterface) (string, error) {
	return p.processWithToolsCommon(p.client, message, tools, toolCaller, display)
}

// ProcessConversationWithUI processes a conversation with tools and UI feedback
func (p *OpenAIProcessor) ProcessConversationWithUI(session *types.ConversationSession, tools []Tool, toolCaller ToolCaller, display ui.ToolDisplayInterface) (string, error) {
	return p.processConversationCommon(p.client, session, tools, toolCaller, display)
}

// Capability methods
func (p *OpenAIProcessor) SupportsConversation() bool {
	return true
}

func (p *OpenAIProcessor) SupportsFunctionCalling() bool {
	return true
}

func (p *OpenAIProcessor) SupportsUIFeedback() bool {
	return true
}

// GetClient returns the underlying LLM client
func (p *OpenAIProcessor) GetClient() LLMClient {
	return p.client
}
