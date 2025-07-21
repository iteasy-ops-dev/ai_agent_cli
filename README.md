# SysEng Agent

AI agent for system engineering tasks with dynamic MCP server and LLM provider management.

## Features

- **Dynamic MCP Server Management**: Add, remove, and monitor Model Context Protocol servers
- **LLM Provider Management**: Support for multiple LLM providers (OpenAI, Anthropic, Google, Local)
- **CLI Interface**: Command-line interface for all operations
- **HTTP API**: RESTful API for programmatic access
- **Real-time Monitoring**: Health checks and status monitoring
- **Configuration Management**: Flexible configuration system

## Installation

```bash
git clone https://github.com/iteasy-ops-dev/syseng-agent
cd syseng-agent
go mod tidy
go build -o syseng-agent main.go
```

## Quick Start

### 1. Add an LLM Provider

```bash
# Add OpenAI provider
./syseng-agent llm add "OpenAI GPT-4" openai gpt-4 --api-key=your-api-key

# Add local LLM
./syseng-agent llm add "Local Llama" local llama2 --endpoint=http://localhost:11434

# Set active provider
./syseng-agent llm set-active <provider-id>
```

### 2. Add MCP Servers

```bash
# Add STDIO MCP server
./syseng-agent mcp add "Local Tools" "/usr/local/bin/mcp-tools" stdio

# Add SSE MCP server
./syseng-agent mcp add "Remote API" "http://api.example.com" sse
```

### 3. Query the Agent

```bash
# Simple query
./syseng-agent agent query "What is the status of the system?"

# Query with specific MCP server
./syseng-agent agent query "Check disk usage" --mcp-server=<server-id>

# Query with specific provider
./syseng-agent agent query "Analyze logs" --provider=<provider-id>
```

### 4. Start Server Mode

```bash
./syseng-agent agent serve --port=8080
```

## API Usage

### Query Endpoint

```bash
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What is the system status?",
    "mcp_server_id": "optional-server-id",
    "provider_id": "optional-provider-id"
  }'
```

### Health Check

```bash
curl http://localhost:8080/api/v1/health
```

### List Resources

```bash
# List MCP servers
curl http://localhost:8080/api/v1/mcp/servers

# List LLM providers
curl http://localhost:8080/api/v1/llm/providers
```

## Configuration

Create a `config.yaml` file:

```yaml
server:
  port: "8080"
  host: "localhost"

database:
  type: "memory"
  path: ""

logging:
  level: "info"
  format: "json"

agent:
  default_provider: ""
  timeout: 30
```

## Commands

### MCP Management

```bash
# List all MCP servers
./syseng-agent mcp list

# Add MCP server
./syseng-agent mcp add <name> <url> <transport>

# Show server details
./syseng-agent mcp show <server-id>

# Remove server
./syseng-agent mcp remove <server-id>
```

### LLM Provider Management

```bash
# List all providers
./syseng-agent llm list

# Add provider
./syseng-agent llm add <name> <type> <model> [flags]

# Show provider details
./syseng-agent llm show <provider-id>

# Set active provider
./syseng-agent llm set-active <provider-id>

# Remove provider
./syseng-agent llm remove <provider-id>
```

### Agent Operations

```bash
# Interactive query
./syseng-agent agent query <message> [flags]

# Start HTTP server
./syseng-agent agent serve [flags]
```

## Supported LLM Providers

- **OpenAI**: GPT-3.5, GPT-4, GPT-4 Turbo
- **Anthropic**: Claude-3, Claude-3.5
- **Google**: Gemini Pro, Gemini Ultra
- **Local**: Ollama, vLLM, any OpenAI-compatible API

## MCP Transport Types

- **STDIO**: Local process communication
- **SSE**: Server-Sent Events over HTTP
- **HTTP**: Standard HTTP requests

## Development

### Building

```bash
go build -o syseng-agent main.go
```

### Testing

```bash
go test ./...
```

### Adding New Features

The project is structured as follows:

- `cmd/`: CLI commands
- `internal/agent/`: Agent logic
- `internal/mcp/`: MCP server management
- `internal/llm/`: LLM provider management
- `internal/config/`: Configuration management
- `internal/logger/`: Logging utilities
- `pkg/types/`: Shared types

## Additional Requirements Validation

Based on the analysis, here are additional components you should consider:

### Security
- **Authentication**: JWT tokens for API access
- **Authorization**: Role-based access control
- **API Key Management**: Secure storage and rotation
- **TLS/HTTPS**: Encrypted communication

### Persistence
- **Database Support**: SQLite, PostgreSQL for persistent storage
- **State Management**: Save/restore server and provider configurations
- **Backup/Restore**: Configuration backup capabilities

### Monitoring & Observability
- **Metrics**: Prometheus metrics integration
- **Tracing**: OpenTelemetry support
- **Alerting**: Health check failures and error notifications
- **Dashboard**: Web UI for monitoring

### Scalability
- **Load Balancing**: Multiple MCP server instances
- **Connection Pooling**: Efficient resource utilization
- **Rate Limiting**: Prevent API abuse
- **Caching**: Response caching for performance

### Integration
- **Plugin System**: Custom MCP server implementations
- **Webhook Support**: Event notifications
- **API Gateway**: Integration with existing infrastructure
- **Container Support**: Docker and Kubernetes deployment

### Error Handling
- **Circuit Breaker**: Fault tolerance patterns
- **Retry Logic**: Automatic retry with backoff
- **Graceful Degradation**: Fallback mechanisms
- **Error Recovery**: Automatic reconnection

Would you like me to implement any of these additional features?