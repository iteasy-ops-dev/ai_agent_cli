package mcp

import (
	"fmt"
	"os"
	"time"

	"github.com/yourusername/syseng-agent/pkg/types"
)

// MockMCPProcess simulates an MCP server for testing
type MockMCPProcess struct {
	server *types.MCPServer
	tools  map[string]Tool
}

func NewMockMCPProcess(server *types.MCPServer) *MockMCPProcess {
	tools := map[string]Tool{
		"list_directory": {
			Name:        "list_directory",
			Description: "List files and directories in a specified path",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory path to list",
						"default":     ".",
					},
				},
			},
		},
		"read_file": {
			Name:        "read_file",
			Description: "Read contents of a file",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path to read",
					},
				},
				"required": []string{"path"},
			},
		},
		"execute_command": {
			Name:        "execute_command",
			Description: "Execute a shell command",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Command to execute",
					},
				},
				"required": []string{"command"},
			},
		},
	}

	return &MockMCPProcess{
		server: server,
		tools:  tools,
	}
}

func (p *MockMCPProcess) Start() error {
	fmt.Printf("Mock MCP server %s started\n", p.server.Name)
	return nil
}

func (p *MockMCPProcess) Stop() error {
	fmt.Printf("Mock MCP server %s stopped\n", p.server.Name)
	return nil
}

func (p *MockMCPProcess) CallTool(name string, arguments map[string]interface{}) (interface{}, error) {
	switch name {
	case "list_directory":
		path := "."
		if pathArg, ok := arguments["path"].(string); ok {
			path = pathArg
		}
		
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}
		
		var files []map[string]interface{}
		for _, entry := range entries {
			info, _ := entry.Info()
			files = append(files, map[string]interface{}{
				"name":    entry.Name(),
				"type":    getFileType(entry),
				"size":    info.Size(),
				"modTime": info.ModTime().Format(time.RFC3339),
			})
		}
		
		return map[string]interface{}{
			"path":  path,
			"files": files,
		}, nil

	case "read_file":
		path, ok := arguments["path"].(string)
		if !ok {
			return nil, fmt.Errorf("path argument is required")
		}
		
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		
		return map[string]interface{}{
			"path":    path,
			"content": string(content),
			"size":    len(content),
		}, nil

	case "execute_command":
		command, ok := arguments["command"].(string)
		if !ok {
			return nil, fmt.Errorf("command argument is required")
		}
		
		// For safety, only allow safe commands in mock
		switch command {
		case "pwd":
			wd, _ := os.Getwd()
			return map[string]interface{}{
				"command": command,
				"output":  wd,
				"status":  0,
			}, nil
		case "date":
			return map[string]interface{}{
				"command": command,
				"output":  time.Now().Format("Mon Jan 2 15:04:05 MST 2006"),
				"status":  0,
			}, nil
		case "whoami":
			return map[string]interface{}{
				"command": command,
				"output":  "syseng-agent-user",
				"status":  0,
			}, nil
		default:
			return map[string]interface{}{
				"command": command,
				"output":  fmt.Sprintf("Mock: would execute '%s'", command),
				"status":  0,
			}, nil
		}

	default:
		return nil, fmt.Errorf("tool %s not found", name)
	}
}

func (p *MockMCPProcess) GetTools() []Tool {
	var tools []Tool
	for _, tool := range p.tools {
		tools = append(tools, tool)
	}
	return tools
}

func getFileType(entry os.DirEntry) string {
	if entry.IsDir() {
		return "directory"
	}
	return "file"
}