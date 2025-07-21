package storage

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
)

type Storage struct {
	dataDir string
}

func New(dataDir string) *Storage {
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".syseng-agent", "data")
	}

	os.MkdirAll(dataDir, 0755)

	return &Storage{
		dataDir: dataDir,
	}
}

func (s *Storage) SaveLLMProviders(providers map[string]*types.LLMProvider) error {
	data, err := json.MarshalIndent(providers, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(s.dataDir, "llm_providers.json")
	return os.WriteFile(filePath, data, 0644)
}

func (s *Storage) LoadLLMProviders() (map[string]*types.LLMProvider, error) {
	filePath := filepath.Join(s.dataDir, "llm_providers.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*types.LLMProvider), nil
		}
		return nil, err
	}

	var providers map[string]*types.LLMProvider
	if err := json.Unmarshal(data, &providers); err != nil {
		return nil, err
	}

	return providers, nil
}

func (s *Storage) SaveMCPServers(servers map[string]*types.MCPServer) error {
	data, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(s.dataDir, "mcp_servers.json")
	return os.WriteFile(filePath, data, 0644)
}

func (s *Storage) LoadMCPServers() (map[string]*types.MCPServer, error) {
	filePath := filepath.Join(s.dataDir, "mcp_servers.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*types.MCPServer), nil
		}
		return nil, err
	}

	var servers map[string]*types.MCPServer
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, err
	}

	return servers, nil
}
