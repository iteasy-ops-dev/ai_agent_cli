package llm

import (
	"fmt"
	"strings"
	"sync"

	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
)

// DefaultClientFactory implements ClientFactory
type DefaultClientFactory struct {
	clients map[string]LLMClient
	mutex   sync.RWMutex
}

// NewDefaultClientFactory creates a new client factory
func NewDefaultClientFactory() *DefaultClientFactory {
	return &DefaultClientFactory{
		clients: make(map[string]LLMClient),
	}
}

// CreateClient creates a client for the given provider
func (f *DefaultClientFactory) CreateClient(provider *types.LLMProvider) (LLMClient, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider cannot be nil")
	}

	// Create a cache key based on provider details
	cacheKey := f.getCacheKey(provider)

	// Check if we already have a client for this provider
	f.mutex.RLock()
	if client, exists := f.clients[cacheKey]; exists {
		f.mutex.RUnlock()
		return client, nil
	}
	f.mutex.RUnlock()

	// Create new client based on provider type
	var client LLMClient
	var err error

	switch strings.ToLower(provider.Type) {
	case "openai":
		client = NewOpenAIClient(provider)
	case "anthropic", "claude":
		client = NewAnthropicClient(provider)
	case "local", "ollama", "llama":
		client = NewLocalClient(provider)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", provider.Type)
	}

	// Validate the client
	if !client.IsHealthy() {
		return nil, fmt.Errorf("client for provider %s is not healthy", provider.Type)
	}

	// Cache the client
	f.mutex.Lock()
	f.clients[cacheKey] = client
	f.mutex.Unlock()

	return client, err
}

// SupportedTypes returns list of supported provider types
func (f *DefaultClientFactory) SupportedTypes() []string {
	return []string{
		"openai",
		"anthropic",
		"claude",
		"local", 
		"ollama",
		"llama",
	}
}

// getCacheKey generates a unique cache key for a provider
func (f *DefaultClientFactory) getCacheKey(provider *types.LLMProvider) string {
	return fmt.Sprintf("%s:%s:%s", provider.Type, provider.Model, provider.Endpoint)
}


// ValidateProvider checks if a provider configuration is valid
func (f *DefaultClientFactory) ValidateProvider(provider *types.LLMProvider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	if provider.Type == "" {
		return fmt.Errorf("provider type is required")
	}

	if provider.Model == "" {
		return fmt.Errorf("provider model is required")
	}

	// Type-specific validation
	switch strings.ToLower(provider.Type) {
	case "openai":
		if provider.APIKey == "" {
			return fmt.Errorf("API key is required for OpenAI provider")
		}
	case "anthropic", "claude":
		if provider.APIKey == "" {
			return fmt.Errorf("API key is required for Anthropic provider")
		}
	case "local", "ollama", "llama":
		if provider.Endpoint == "" {
			return fmt.Errorf("endpoint is required for local provider")
		}
	default:
		return fmt.Errorf("unsupported provider type: %s", provider.Type)
	}

	return nil
}

