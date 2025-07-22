package llm

import (
	"fmt"
	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
	"github.com/iteasy-ops-dev/syseng-agent/internal/ui"
)

// ToolCaller defines the function signature for tool execution
type ToolCaller func(name string, args map[string]interface{}) (interface{}, error)

// LLMProcessor defines the interface for processing LLM requests with different strategies
type LLMProcessor interface {
	// ProcessWithTools processes a simple message with available tools
	ProcessWithTools(message string, tools []Tool, toolCaller ToolCaller) (string, error)
	
	// ProcessConversation processes a message within a conversation context
	ProcessConversation(session *types.ConversationSession, tools []Tool, toolCaller ToolCaller) (string, error)
	
	// ProcessWithUI processes a message with UI feedback support
	ProcessWithUI(message string, tools []Tool, toolCaller ToolCaller, display ui.ToolDisplayInterface) (string, error)
	
	// ProcessConversationWithUI processes a conversation with UI feedback support
	ProcessConversationWithUI(session *types.ConversationSession, tools []Tool, toolCaller ToolCaller, display ui.ToolDisplayInterface) (string, error)
	
	// Capability queries
	SupportsConversation() bool
	SupportsFunctionCalling() bool
	SupportsUIFeedback() bool
	
	// Provider information
	GetProviderType() string
	GetProviderModel() string
	
	// Client access
	GetClient() LLMClient
}

// BaseProcessor provides common functionality for all LLM processors
type BaseProcessor struct {
	provider      *types.LLMProvider
	promptManager PromptManager
}

// NewBaseProcessor creates a new BaseProcessor
func NewBaseProcessor(provider *types.LLMProvider, promptManager PromptManager) *BaseProcessor {
	return &BaseProcessor{
		provider:      provider,
		promptManager: promptManager,
	}
}

// GetProviderType returns the LLM provider type
func (bp *BaseProcessor) GetProviderType() string {
	return bp.provider.Type
}

// GetProviderModel returns the LLM provider model
func (bp *BaseProcessor) GetProviderModel() string {
	return bp.provider.Model
}

// buildEnhancedMessage creates an enhanced message with tool context
func (bp *BaseProcessor) buildEnhancedMessage(baseMessage string, toolCount int) string {
	return EnhanceMessageWithTools(baseMessage, toolCount)
}

// getSystemPrompt returns the system prompt for this processor's LLM type
func (bp *BaseProcessor) getSystemPrompt() string {
	return bp.promptManager.GetSystemPrompt(bp.provider.Type)
}

// getConversationPrompt returns the conversation prompt for this processor's LLM type
func (bp *BaseProcessor) getConversationPrompt() string {
	return bp.promptManager.GetConversationPrompt(bp.provider.Type)
}

// processWithToolsCommon provides common tool processing logic
func (bp *BaseProcessor) processWithToolsCommon(client LLMClient, message string, tools []Tool, toolCaller ToolCaller, display ui.ToolDisplayInterface) (string, error) {
	enhancedMessage := bp.buildEnhancedMessage(message, len(tools))
	
	// Use UI wrapper if display is provided
	var wrappedToolCaller ToolCaller
	if display != nil && toolCaller != nil {
		wrappedToolCaller = WrapToolCallerWithUI(toolCaller, display)
	} else {
		wrappedToolCaller = toolCaller
	}
	
	if len(tools) == 0 {
		if display != nil {
			display.ShowProgress(fmt.Sprintf(ProgressProcessingWithLLM, bp.provider.Type))
		}
		return client.ProcessMessage(enhancedMessage)
	}
	
	if display != nil {
		display.ShowProgress(fmt.Sprintf(ProgressProcessingWithTools, bp.provider.Type))
	}
	
	// Try to use tool support if available
	if toolSupport, ok := client.(ToolSupport); ok {
		result, err := toolSupport.ProcessWithTools(enhancedMessage, tools, wrappedToolCaller)
		if err != nil && display != nil {
			display.ShowError(err)
		}
		return result, err
	}
	
	// Fallback to basic processing
	return client.ProcessMessage(enhancedMessage)
}

// processConversationCommon provides common conversation processing logic
func (bp *BaseProcessor) processConversationCommon(client LLMClient, session *types.ConversationSession, tools []Tool, toolCaller ToolCaller, display ui.ToolDisplayInterface) (string, error) {
	// Use UI wrapper if display is provided
	var wrappedToolCaller ToolCaller
	if display != nil && toolCaller != nil {
		wrappedToolCaller = WrapToolCallerWithUI(toolCaller, display)
	} else {
		wrappedToolCaller = toolCaller
	}
	
	if len(tools) == 0 {
		if display != nil {
			display.ShowProgress(fmt.Sprintf(ProgressProcessingConversation, bp.provider.Type))
		}
		if conversationSupport, ok := client.(ConversationSupport); ok {
			return conversationSupport.ProcessConversation(session)
		}
		// Fallback to simple message processing
		lastMessage := session.GetLastUserMessage()
		return client.ProcessMessage(lastMessage)
	}
	
	if display != nil {
		display.ShowProgress(fmt.Sprintf(ProgressProcessingConversationWithTools, bp.provider.Type))
	}
	
	// Try conversation with tools
	if conversationSupport, ok := client.(ConversationSupport); ok {
		result, err := conversationSupport.ProcessConversationWithTools(session, tools, wrappedToolCaller)
		if err != nil && display != nil {
			display.ShowError(err)
		}
		return result, err
	}
	
	// Fallback to tool processing with last message
	lastMessage := session.GetLastUserMessage()
	return bp.processWithToolsCommon(client, lastMessage, tools, toolCaller, display)
}

// Common capability implementations
func (bp *BaseProcessor) SupportsUIFeedback() bool {
	return true // All processors support UI feedback through common base
}

// ProcessorFactory creates LLM processors based on provider type
type ProcessorFactory interface {
	CreateProcessor(provider *types.LLMProvider) (LLMProcessor, error)
}

// DefaultProcessorFactory implements ProcessorFactory
type DefaultProcessorFactory struct {
	promptManager PromptManager
}

// NewDefaultProcessorFactory creates a new DefaultProcessorFactory
func NewDefaultProcessorFactory() *DefaultProcessorFactory {
	return &DefaultProcessorFactory{
		promptManager: NewDefaultPromptManager(),
	}
}

// CreateProcessor creates an appropriate processor for the given provider
func (f *DefaultProcessorFactory) CreateProcessor(provider *types.LLMProvider) (LLMProcessor, error) {
	switch provider.Type {
	case ProviderOpenAI:
		return NewOpenAIProcessor(provider, f.promptManager), nil
	case ProviderAnthropic:
		return NewAnthropicProcessor(provider, f.promptManager), nil
	case ProviderLocal:
		return NewLocalProcessor(provider, f.promptManager), nil
	default:
		// Default to OpenAI processor for unknown types
		return NewOpenAIProcessor(provider, f.promptManager), nil
	}
}

// SetPromptManager allows updating the prompt manager
func (f *DefaultProcessorFactory) SetPromptManager(pm PromptManager) {
	f.promptManager = pm
}