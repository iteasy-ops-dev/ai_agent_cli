package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/iteasy-ops-dev/syseng-agent/internal/storage"
	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
)

type Manager struct {
	providers map[string]*types.LLMProvider
	storage   *storage.Storage
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	storage := storage.New("")

	m := &Manager{
		providers: make(map[string]*types.LLMProvider),
		storage:   storage,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Load existing providers from storage
	if providers, err := storage.LoadLLMProviders(); err == nil {
		m.providers = providers
	}

	return m
}

func (m *Manager) AddProvider(provider *types.LLMProvider) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if provider.ID == "" {
		provider.ID = uuid.New().String()
	}

	provider.CreatedAt = time.Now()
	provider.UpdatedAt = time.Now()

	if err := m.validateProvider(provider); err != nil {
		return fmt.Errorf("invalid provider: %w", err)
	}

	m.providers[provider.ID] = provider

	// Save to storage
	if err := m.storage.SaveLLMProviders(m.providers); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to save providers to storage: %v\n", err)
	}

	return nil
}

func (m *Manager) RemoveProvider(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.providers[id]; !exists {
		return fmt.Errorf("provider %s not found", id)
	}

	delete(m.providers, id)

	// Save to storage
	if err := m.storage.SaveLLMProviders(m.providers); err != nil {
		fmt.Printf("Warning: failed to save providers to storage: %v\n", err)
	}

	return nil
}

func (m *Manager) GetProvider(id string) (*types.LLMProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[id]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", id)
	}

	return provider, nil
}

func (m *Manager) ListProviders() []*types.LLMProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]*types.LLMProvider, 0, len(m.providers))
	for _, provider := range m.providers {
		providers = append(providers, provider)
	}

	return providers
}

func (m *Manager) GetActiveProvider() (*types.LLMProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, provider := range m.providers {
		if provider.IsActive {
			return provider, nil
		}
	}

	return nil, fmt.Errorf("no active provider found")
}

func (m *Manager) SetActiveProvider(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	provider, exists := m.providers[id]
	if !exists {
		return fmt.Errorf("provider %s not found", id)
	}

	for _, p := range m.providers {
		p.IsActive = false
		p.UpdatedAt = time.Now()
	}

	provider.IsActive = true
	provider.UpdatedAt = time.Now()

	// Save to storage
	if err := m.storage.SaveLLMProviders(m.providers); err != nil {
		fmt.Printf("Warning: failed to save providers to storage: %v\n", err)
	}

	return nil
}

func (m *Manager) UpdateProvider(id string, updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	provider, exists := m.providers[id]
	if !exists {
		return fmt.Errorf("provider %s not found", id)
	}

	if name, ok := updates["name"].(string); ok {
		provider.Name = name
	}

	if endpoint, ok := updates["endpoint"].(string); ok {
		provider.Endpoint = endpoint
	}

	if model, ok := updates["model"].(string); ok {
		provider.Model = model
	}

	if config, ok := updates["config"].(map[string]interface{}); ok {
		provider.Config = config
	}

	provider.UpdatedAt = time.Now()

	if err := m.validateProvider(provider); err != nil {
		return err
	}

	// Save to storage
	if err := m.storage.SaveLLMProviders(m.providers); err != nil {
		fmt.Printf("Warning: failed to save providers to storage: %v\n", err)
	}

	return nil
}

func (m *Manager) validateProvider(provider *types.LLMProvider) error {
	if provider.Name == "" {
		return fmt.Errorf("provider name is required")
	}

	if provider.Type == "" {
		return fmt.Errorf("provider type is required")
	}

	switch provider.Type {
	case "openai", "anthropic", "google", "local":
		// Valid types
	default:
		return fmt.Errorf("unsupported provider type: %s", provider.Type)
	}

	if provider.Type != "local" && provider.APIKey == "" {
		return fmt.Errorf("API key is required for provider type %s", provider.Type)
	}

	if provider.Type == "local" {
		if provider.Endpoint == "" {
			return fmt.Errorf("endpoint is required for local provider")
		}
		
		// Auto-correct common Ollama endpoint configurations
		provider.Endpoint = m.normalizeLocalEndpoint(provider.Endpoint)
	}

	return nil
}

// normalizeLocalEndpoint auto-corrects common Ollama endpoint issues
func (m *Manager) normalizeLocalEndpoint(endpoint string) string {
	// Remove trailing slash
	endpoint = strings.TrimSuffix(endpoint, "/")
	
	// Auto-detect and fix common Ollama configurations
	if strings.Contains(endpoint, ":11434") {
		// This looks like an Ollama endpoint
		if !strings.HasSuffix(endpoint, "/api/chat") {
			// Add the correct Ollama chat endpoint path
			fmt.Printf("Info: Auto-correcting Ollama endpoint from '%s' to '%s/api/chat'\n", endpoint, endpoint)
			endpoint = endpoint + "/api/chat"
		}
	} else if strings.Contains(endpoint, "ollama") && !strings.Contains(endpoint, "/api/chat") {
		// Generic ollama endpoint without the chat path
		fmt.Printf("Info: Adding /api/chat to Ollama endpoint: %s\n", endpoint)
		endpoint = endpoint + "/api/chat"
	}
	
	return endpoint
}

func (m *Manager) Shutdown() {
	m.cancel()
}
