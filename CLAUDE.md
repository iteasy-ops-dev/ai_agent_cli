# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Essential Commands

### Build and Run
```bash
# Build the application
go build -o syseng-agent .

# Cross-platform builds
./build.sh                 # Build for all platforms (Unix/macOS)
build.bat                  # Build for all platforms (Windows)
make build-all             # Build for all platforms using Makefile

# Platform-specific builds
make build-windows         # Windows (amd64, 386)
make build-macos           # macOS (Intel, Apple Silicon)
make build-linux           # Linux (amd64, arm64, 386)

# Development builds
make build                 # Current platform only
make dev                   # Build and run

# Clean builds
make clean
./build.sh clean

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

# NEW: Interactive chat session (basic mode)
./syseng-agent chat

# NEW: Interactive chat with TUI interface (enhanced mode)
./syseng-agent chat --tui

# Interactive chat with tool approval
./syseng-agent chat -i --tui

# Debug mode (shows detailed MCP server health and tool loading info)
DEBUG=1 ./syseng-agent chat
DEBUG=1 ./syseng-agent agent query "test message"

# List current resources
./syseng-agent mcp list
./syseng-agent llm list
```

### New Chat Features (Interactive Conversation Mode)

**Chat Commands**:
- `./syseng-agent chat` - Basic chat mode with readline interface
- `./syseng-agent chat --tui` - Enhanced TUI mode with Bubble Tea
- `./syseng-agent chat -i` - Chat with interactive tool approval
- `./syseng-agent chat --tui -i` - Full-featured TUI with tool approval

**TUI Chat Interface** (Terminal User Interface):
- 📝 Multi-line textarea for composing messages
- 📜 Scrollable conversation history viewport
- 🎨 Syntax-highlighted tool calls and results
- ⌨️ Keyboard shortcuts: Ctrl+Enter (send), Ctrl+L (clear), Esc (quit)
- 🔧 Real-time tool execution display
- 📊 Execution summaries with timing information

**Basic Chat Mode**:
- Simple readline-based conversation loop
- Multi-line input support with `\n` escape sequences
- Built-in commands: help, clear, exit, quit
- Maintains conversation context within session

### Enhanced UI Features (Tool Execution Display)

**Interactive Mode** (`-i` or `--interactive`):
- User approval required for each tool execution
- Real-time tool call display with parameters
- Options: [y]es, [n]o, [s]kip prompts (auto-approve all), [a]bort
- Prevents unwanted system modifications
- 's' option changed from "skip all" to "auto-approve all remaining tools"

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

### New Interface-Based Design (Refactored Architecture)

The system has been completely refactored using modern design patterns for better extensibility and maintainability:

**Design Patterns Implementation:**
- **Strategy Pattern**: `LLMProcessor` interface with provider-specific implementations
- **Factory Pattern**: `ClientFactory` creates appropriate clients based on provider type
- **Interface Segregation**: Modular capabilities through `ToolSupport` and `ConversationSupport`
- **Template Method**: `BaseProcessor` provides common functionality for all processors

### LLM Processing Flow (New Architecture)

```
User Query → Agent → LLMProcessor → LLMClient → Provider API
                ↓
            MCP Tools ← Tool Execution ← Tool Calling
```

**Interface Hierarchy:**
```go
LLMClient (base interface)
├── ProcessMessage(message string) (string, error)
├── GetProviderInfo() ProviderInfo
└── IsHealthy() bool

ToolSupport (extends LLMClient)
├── ProcessWithTools(message, tools, toolCaller) (string, error)
└── SupportsFunctionCalling() bool

ConversationSupport (extends LLMClient)
├── ProcessConversation(session) (string, error)
├── ProcessConversationWithTools(session, tools, toolCaller) (string, error)
└── SupportsConversation() bool
```

### Key Components (Refactored)

**LLMProcessor Interface** (`internal/llm/processor.go`)
- Defines processing strategies for different LLM providers
- Implements Strategy Pattern for provider-specific behavior
- Methods: `ProcessWithTools()`, `ProcessConversation()`, `ProcessWithUI()`, `ProcessConversationWithUI()`

**LLMClient Implementations**:
- **OpenAIClient** (`internal/llm/openai_client.go`): Full ToolSupport and ConversationSupport
- **AnthropicClient** (`internal/llm/anthropic_client.go`): Basic tool support via prompt engineering  
- **LocalClient** (`internal/llm/local_client.go`): OpenAI-compatible local models

**ClientFactory** (`internal/llm/client_factory.go`)
- Creates appropriate clients based on provider configuration
- Includes caching and validation functionality
- Supports extensibility for new providers

**PromptManager** (`internal/llm/prompts.go`)
- Centralized prompt template management
- Provider-specific prompt customization
- System prompts with tool availability context
- Error handling and fallback strategies

**Simplified Agent Flow** (`internal/agent/agent.go`)
- Uses new processor/client system instead of direct LLM calls
- Removed redundant `process*` functions
- Single `prepareMCPTools()` helper function
- Enhanced with UI feedback support

### Legacy vs New Architecture

**REMOVED (Legacy)**:
- `CallOpenAIWithTools()`, `CallAnthropicWithTools()`, `CallLocalWithTools()`
- `processOpenAIWithMCP()`, `processAnthropicWithMCP()`, `processLocalWithMCP()`
- Duplicate `getString()` and `getMap()` functions across files

**NEW (Refactored)**:
- Interface-based client system with capability detection
- Strategy Pattern processors for each provider
- Factory Pattern for client creation
- Centralized prompt management
- Consolidated utilities in `pkg/utils/`

### MCP Manager (Enhanced)
- **stdio servers**: Skip health checks (no persistent connection needed)
- **SSE/HTTP servers**: Maintain persistent connections with health monitoring
- **Tool discovery**: Cached in `server.Tools` with full schema information
- **LastPing updates**: Successful tool calls update server health timestamps
- **Process storage**: Thread-safe operations with proper mutex handling

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

## Development Patterns (New Architecture)

### Adding New LLM Provider Support

**1. Create Client Implementation**:
```go
// Implement base LLMClient interface
type NewProviderClient struct {
    provider *types.LLMProvider
    apiKey   string
    baseURL  string
}

func (c *NewProviderClient) ProcessMessage(message string) (string, error) {
    // Basic message processing logic
}

func (c *NewProviderClient) GetProviderInfo() ProviderInfo {
    return ProviderInfo{
        Name:    c.provider.Name,
        Type:    c.provider.Type,
        Model:   c.provider.Model,
        Healthy: c.IsHealthy(),
    }
}

func (c *NewProviderClient) IsHealthy() bool {
    // Health check logic
}
```

**2. Implement Optional Capabilities**:
```go
// Add ToolSupport if provider supports function calling
func (c *NewProviderClient) ProcessWithTools(message string, tools []Tool, toolCaller ToolCaller) (string, error) {
    // Convert tools to provider format
    // Implement conversation loop with tool execution
    // Return final response
}

func (c *NewProviderClient) SupportsFunctionCalling() bool {
    return true // or false based on capabilities
}

// Add ConversationSupport if provider supports multi-turn conversations
func (c *NewProviderClient) ProcessConversation(session *types.ConversationSession) (string, error) {
    // Convert conversation history to provider format
    // Process with conversation context
}

func (c *NewProviderClient) SupportsConversation() bool {
    return true // or false based on capabilities
}
```

**3. Create Processor Implementation**:
```go
type NewProviderProcessor struct {
    *BaseProcessor
    client LLMClient
}

func NewNewProviderProcessor(provider *types.LLMProvider, promptManager PromptManager) *NewProviderProcessor {
    client := NewNewProviderClient(provider)
    return &NewProviderProcessor{
        BaseProcessor: NewBaseProcessor(provider, promptManager),
        client:        client,
    }
}

// ProcessWithTools delegates to client with UI enhancements
func (p *NewProviderProcessor) ProcessWithTools(message string, tools []Tool, toolCaller ToolCaller) (string, error) {
    enhancedMessage := p.buildEnhancedMessage(message, len(tools))
    
    if toolSupport, ok := p.client.(ToolSupport); ok {
        return toolSupport.ProcessWithTools(enhancedMessage, tools, toolCaller)
    }
    
    // Fallback to basic processing
    return p.client.ProcessMessage(enhancedMessage)
}
```

**4. Update Factory**:
```go
// Add case to client_factory.go
func (f *ClientFactory) CreateClient(provider *types.LLMProvider) (LLMClient, error) {
    switch provider.Type {
    case "newprovider":
        return NewNewProviderClient(provider), nil
    // existing cases...
    }
}
```

**5. Add Processor Factory**:
```go
// Add case to processor factory
func NewLLMProcessor(provider *types.LLMProvider, promptManager PromptManager) LLMProcessor {
    switch provider.Type {
    case "newprovider":
        return NewNewProviderProcessor(provider, promptManager)
    // existing cases...
    }
}
```

### Adding New MCP Server Support
1. Test with stdio transport first (most reliable)
2. Ensure tool discovery completes before marking as "available"
3. Store tools in both `server.Capabilities` (names) and `server.Tools` (full schema)
4. Handle process lifecycle: start → discover → store → health monitor
5. Use appropriate health check strategy based on transport type

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

## Project Structure Insights (Refactored)

**CLI Layer** (`cmd/`): Cobra commands with Viper config management
**Internal Layer** (`internal/`): Core business logic, not importable externally
**Types Layer** (`pkg/types/`): Shared data structures across all components
**Utils Layer** (`pkg/utils/`): Common utility functions (GetString, GetMap)

**Key Integration Points (New Architecture)**:
- `agent.go:ProcessRequestWithUI()` - Main LLM-MCP orchestration using processors
- `processor.go:LLMProcessor` - Strategy interface for provider-specific processing
- `client_interface.go:LLMClient` - Base interface with optional capabilities
- `client_factory.go:CreateClient()` - Factory for creating appropriate clients
- `prompts.go:PromptManager` - Centralized prompt template management
- `manager.go:testStdioServer()` - MCP server validation and tool caching
- `protocol.go:MCPProcess` - JSON-RPC communication handling

**Removed Legacy Integration Points**:
- ~~`agent.go:processOpenAIWithMCP()`~~ - Replaced with processor pattern
- ~~`clients.go:CallOpenAIWithTools()`~~ - Functionality moved to individual clients
- ~~`clients.go:CallAnthropicWithTools()`~~ - Replaced with AnthropicClient
- ~~`clients.go:CallLocalWithTools()`~~ - Replaced with LocalClient

**New File Organization** (`internal/llm/`):
```
llm/
├── client_interface.go     # Core interfaces (LLMClient, ToolSupport, ConversationSupport)
├── client_factory.go       # Factory Pattern for client creation
├── prompts.go             # Centralized prompt management
├── processor.go           # Strategy Pattern interface
├── openai_client.go       # OpenAI-specific client implementation
├── openai_processor.go    # OpenAI-specific processor implementation
├── anthropic_client.go    # Anthropic-specific client implementation
├── anthropic_processor.go # Anthropic-specific processor implementation
├── local_client.go        # Local LLM client implementation
├── local_processor.go     # Local LLM processor implementation
├── clients.go             # Type definitions and conversion utilities
└── manager.go             # Provider management (existing)
```

## Build System

### Cross-Platform Support
The project includes comprehensive build scripts for multiple platforms:

**Supported Platforms:**
- Windows: amd64, 386
- macOS: amd64 (Intel), arm64 (Apple Silicon)
- Linux: amd64, arm64, 386

**Build Output Structure:**
```
dist/
├── windows/
│   ├── syseng-agent-windows-amd64.exe
│   ├── syseng-agent-windows-amd64.exe.zip
│   └── syseng-agent-windows-386.exe
├── macos/
│   ├── syseng-agent-macos-amd64
│   ├── syseng-agent-macos-amd64.tar.gz
│   └── syseng-agent-macos-arm64
├── linux/
│   ├── syseng-agent-linux-amd64
│   ├── syseng-agent-linux-amd64.tar.gz
│   └── syseng-agent-linux-arm64
└── checksums.txt
```

**Build Features:**
- Automatic archive creation (ZIP for Windows, tar.gz for Unix)
- SHA256 checksum generation
- Version injection with build time and git commit
- Colored output and progress indicators
- Clean build directory management

**Version Information:**
```bash
./syseng-agent --version
# Output:
# syseng-agent v1.0.0
# Built: 2024-01-15T10:30:00Z
# Commit: abc1234
```

### Error Handling Improvements
Enhanced LLM prompt engineering for better error recovery:
- System prompt includes fallback strategies
- Common error hints provided to LLM
- Alternative path suggestions for file operations
- Never give up after single failure approach

### Tool Calling Improvements
Fixed and enhanced tool calling behavior:
- **Aggressive tool usage**: System prioritizes tool usage for any system operation
- **Clear examples**: "ping google" → automatically uses start_process tool
- **Debug visibility**: Shows tool loading status and count during processing
- **Context awareness**: Uses conversation history for follow-up questions
- **Fallback prevention**: Never says "no capability" when tools are available

### MCP Server Health Check Fixes
Fixed critical issue where MCP servers became "unhealthy" after tool use:
- **LastPing Updates**: Successful tool calls now update server.LastPing timestamp
- **Transport-Aware Health Checks**: Different timeouts for stdio (5min) vs persistent (60s) servers
- **Debug Logging**: Enhanced debug output for health check decisions and LastPing updates
- **Root Cause**: stdio servers start fresh processes per tool call but weren't updating health status

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

**Build Prerequisites:**
- Go 1.21+
- Git (for commit information)
- tar (Unix/Linux for archives)
- PowerShell (Windows for ZIP creation)
- make (optional, for Makefile usage)