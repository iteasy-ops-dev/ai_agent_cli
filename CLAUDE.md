# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Essential Commands

### Build and Run
```bash
# Build the application
go build -o syseng-agent .

# Run tests
go test ./...

# Test specific package
go test ./internal/mcp
go test ./internal/llm

# Build and test with dependencies
go mod tidy && go build -o syseng-agent .
```

### Development Workflow
```bash
# Add MCP server (most common during development)
./syseng-agent mcp add "Desktop Commander" "npx -y @wonderwhy-er/desktop-commander@latest" stdio

# Add LLM provider
./syseng-agent llm add "OpenAI GPT-4" openai gpt-4 --api-key=your-key

# Test agent integration (standard mode)
./syseng-agent agent query "test message"

# Test agent with interactive mode (NEW! User can approve each tool)
./syseng-agent agent query -i "check system information"

# List current resources
./syseng-agent mcp list
./syseng-agent llm list
```

### New UI Features (Enhanced User Interface)

**Interactive Mode** (`-i` or `--interactive`):
- User approval required for each tool execution
- Real-time tool call display with parameters
- Options: [y]es, [n]o, [s]kip all, [a]bort
- Prevents unwanted system modifications

**Visual Enhancements**:
- 🔧 Color-coded tool calls (blue)
- ✅ Success indicators (green) 
- ❌ Error displays (red)
- ⏳ Progress indicators (yellow)
- ⏱️ Execution timing information

**Tool Execution Flow**:
```
⏱️ [+0.1s] ⏳ Initializing AI agent...
⏱️ [+0.2s] ⏳ Finding LLM provider...
⏱️ [+0.5s] ⏳ Processing with LLM and available tools...
🔧 Tool Call: Desktop_Commander.start_process (command=uname -a)
✅ Success (0.45s)
   Darwin MacBook-Pro.local 21.6.0 Darwin Kernel Version...

📊 Execution Summary
   Total tools called: 3
   Successful: 3, Failed: 0
   Total duration: 1.2s
   Tool execution order:
     1. ✅ Desktop_Commander.start_process (0.45s)
     2. ✅ Desktop_Commander.read_file (0.32s)
     3. ✅ Desktop_Commander.list_directory (0.28s)
```

## Core Architecture

### LLM-MCP Integration Flow
The heart of this system is the **OpenAI Function Calling ↔ MCP Tools** integration:

1. **User Query** → `internal/agent/agent.go`
2. **Tool Discovery** → `internal/mcp/manager.go:GetAllTools()`
3. **Function Conversion** → `internal/llm/clients.go:ConvertMCPToolsToOpenAI()`
4. **Iterative LLM Calls** → `internal/llm/clients.go:CallOpenAIWithTools()`
5. **Tool Execution** → `internal/mcp/manager.go:CallTool()`

### Key Components

**CallOpenAIWithTools Function** (`internal/llm/clients.go`)
- Implements the conversation loop between LLM and MCP tools
- Manages message context across multiple tool calls
- Handles up to 10 iterations for complex multi-tool workflows
- Core pattern: API call → tool detection → tool execution → continue loop

**MCP Manager** (`internal/mcp/manager.go`)
- **stdio servers**: Start fresh process per tool call, store tool metadata
- **SSE/HTTP servers**: Maintain persistent connections
- **Tool discovery**: Happens once at server registration, cached in `server.Tools`
- **Process storage**: Uses mutex-free approach for stdio to avoid deadlocks

**Agent Orchestrator** (`internal/agent/agent.go`)
- Combines LLM providers with MCP tools
- Enhances user messages with tool availability context
- Handles tool name cleaning for OpenAI compatibility (spaces → underscores)
- **NEW**: `ProcessRequestWithUI()` method for enhanced user experience

**UI Package** (`internal/ui/`)
- **ToolDisplayInterface**: Pluggable display system for different interaction modes
- **InteractiveDisplay**: Prompts user for tool execution approval
- **NonInteractiveDisplay**: Shows tool execution without interruption
- **SpinnerDisplay**: Wrapper adding animated progress indicators
- **TimedProgressDisplay**: Wrapper adding execution timing information
- **Color utilities**: ANSI color support with NO_COLOR environment variable respect

### MCP Protocol Implementation

**Transport Types**:
- **stdio**: JSON-RPC over stdin/stdout (most common, used for desktop-commander)
- **SSE**: Server-Sent Events over HTTP
- **HTTP**: Standard HTTP requests

**Critical Implementation Details**:
- stdio servers start via `strings.Fields(server.URL)` command parsing
- Tool discovery uses JSON-RPC 2.0: `initialize` → `tools/list` → store results
- Error filtering in `readErrors()` hides normal loading messages
- OpenAI Function names must match `^[a-zA-Z0-9_-]+$` pattern

### Storage and State Management

**Persistence**: JSON files via `internal/storage/storage.go`
- MCP servers with discovered tools cached in `server.Tools`
- LLM providers with `APIKey` field properly serialized
- No database required, file-based storage

**Concurrency**: 
- RWMutex for server/provider maps
- Mutex-free stdio server registration to prevent deadlocks
- Process maps store MCPProcessInterface implementations

## Development Patterns

### Adding New MCP Server Support
1. Test with stdio transport first (most reliable)
2. Ensure tool discovery completes before marking as "available"
3. Store tools in both `server.Capabilities` (names) and `server.Tools` (full schema)
4. Handle process lifecycle: start → discover → store → health monitor

### LLM Provider Integration
```go
// Must support OpenAI Function Calling for tool integration
func CallProviderWithTools(provider, message, tools, toolCaller) {
    // Convert MCP tools to provider's function format
    // Implement conversation loop with tool execution
    // Return final response after tool chain completion
}
```

### Tool Naming and Compatibility
- MCP tool names: `server_name_tool_name` format
- Clean server names: replace spaces/special chars with underscores
- OpenAI functions: strict alphanumeric + underscore/hyphen only

### UI Integration Patterns
```go
// Create layered display with multiple features
base := ui.NewInteractiveDisplay()
display := ui.NewTimedProgressDisplay(ui.NewSpinnerDisplay(base))

// Tool execution with user feedback
display.ShowToolCall(serverName, toolName, args)
approved, err := display.PromptToolApproval(serverName, toolName, args)
if approved {
    result, err := executeToolCall()
    display.ShowToolResult(result, duration)
}
display.ShowSummary(executionSummary)
```

**Display Wrapper Pattern**: Each UI enhancement is a wrapper around the base interface, allowing modular composition of features (spinners + timing + colors + interaction).

## Project Structure Insights

**CLI Layer** (`cmd/`): Cobra commands with Viper config management
**Internal Layer** (`internal/`): Core business logic, not importable externally
**Types Layer** (`pkg/types/`): Shared data structures across all components

**Key Integration Points**:
- `agent.go:processOpenAIWithMCP()` - Main LLM-MCP orchestration
- `clients.go:CallOpenAIWithTools()` - Conversation loop implementation  
- `manager.go:testStdioServer()` - MCP server validation and tool caching
- `protocol.go:MCPProcess` - JSON-RPC communication handling

## Configuration

Default config location: `config.yaml`
```yaml
server:
  port: "8080"
  host: "localhost"
logging:
  level: "info"
```

**Environment**: Go 1.21+, requires network access for LLM APIs and MCP server downloads.