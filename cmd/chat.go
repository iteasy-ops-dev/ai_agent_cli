package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/iteasy-ops-dev/syseng-agent/internal/agent"
	"github.com/iteasy-ops-dev/syseng-agent/internal/tui"
	"github.com/iteasy-ops-dev/syseng-agent/internal/ui"
	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session with the AI agent",
	Long: `Start an interactive chat session with the AI agent.
This provides a conversational interface similar to Claude Code where you can
send multiple messages and maintain conversation context.

Use 'exit', 'quit', or Ctrl+C to end the session.`,
	Run: func(cmd *cobra.Command, args []string) {
		mcpServerID, _ := cmd.Flags().GetString("mcp-server")
		providerID, _ := cmd.Flags().GetString("provider")
		interactive, _ := cmd.Flags().GetBool("interactive")
		tui, _ := cmd.Flags().GetBool("tui")

		if tui {
			startTUIChat(mcpServerID, providerID, interactive)
		} else {
			startBasicChat(mcpServerID, providerID, interactive)
		}
	},
}

func startBasicChat(mcpServerID, providerID string, interactive bool) {
	fmt.Println("🤖 Starting chat session with AI agent...")
	fmt.Println("Type 'exit', 'quit', or press Ctrl+C to end the session.")
	fmt.Println("Type 'help' for available commands.")
	fmt.Println("Use '\\n' in your message for line breaks.")
	fmt.Println(strings.Repeat("=", 60))

	ag := agent.New(mcpManager, llmManager)
	scanner := bufio.NewScanner(os.Stdin)

	// Create conversation session
	session := &types.ConversationSession{
		ID:          uuid.New().String(),
		MCPServerID: mcpServerID,
		ProviderID:  providerID,
		Interactive: interactive,
		Messages:    []types.ConversationMessage{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create display interface
	var display ui.ToolDisplayInterface
	if interactive {
		base := ui.NewInteractiveDisplay()
		// Use only timed progress, no spinner to avoid hanging
		display = ui.NewTimedProgressDisplay(base)
	} else {
		base := ui.NewNonInteractiveDisplay()
		// Use only timed progress, no spinner to avoid hanging
		display = ui.NewTimedProgressDisplay(base)
	}

	for {
		fmt.Print("\n💬 You: ")
		
		if !scanner.Scan() {
			break // EOF or error
		}

		input := strings.TrimSpace(scanner.Text())
		
		// Handle special commands
		switch strings.ToLower(input) {
		case "exit", "quit", "q":
			fmt.Println("👋 Goodbye!")
			return
		case "help", "h":
			showChatHelp()
			continue
		case "clear", "cls":
			clearScreen()
			// Reset conversation session
			session = &types.ConversationSession{
				ID:          uuid.New().String(),
				MCPServerID: mcpServerID,
				ProviderID:  providerID,
				Interactive: interactive,
				Messages:    []types.ConversationMessage{},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			continue
		case "":
			continue // Skip empty input
		}

		// Process multi-line input (simple implementation)
		message := strings.ReplaceAll(input, "\\n", "\n")

		fmt.Printf("🔄 Processing your request...\n")

		// Check if we can use streaming
		var provider *types.LLMProvider
		var err error
		if session.ProviderID != "" {
			provider, err = llmManager.GetProvider(session.ProviderID)
		} else {
			provider, err = llmManager.GetActiveProvider()
		}
		
		if err == nil && provider != nil {
			// Try streaming if supported
			if streamSupported := checkStreamingSupport(provider); streamSupported {
				// Update session with provider ID if not set
				if session.ProviderID == "" && provider != nil {
					session.ProviderID = provider.ID
				}
				
				fmt.Printf("🔄 Using streaming for provider: %s (%s)\n", provider.Name, provider.Type)
				err = processWithStreaming(ag, session, message, display)
				if err != nil {
					fmt.Printf("❌ Streaming error: %v\n", err)
					fmt.Printf("🔄 Falling back to non-streaming...\n")
					// Don't continue here, fall through to non-streaming
				} else {
					continue
				}
			}
		}

		// Fall back to non-streaming
		response, err := ag.ProcessConversation(session, message, display)
		
		// Clear any remaining progress indicators by printing newline
		fmt.Print("\r\033[K")  // Clear current line
		
		if err != nil {
			fmt.Printf("❌ Error processing request: %v\n", err)
			continue
		}

		fmt.Printf("\n🤖 Agent: %s\n", response.Message)

		if response.Error != "" {
			fmt.Printf("⚠️  Warning: %s\n", response.Error)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}

func startTUIChat(mcpServerID, providerID string, interactive bool) {
	ag := agent.New(mcpManager, llmManager)
	
	fmt.Println("🚀 Starting TUI chat interface...")
	err := tui.StartTUIChat(ag, mcpServerID, providerID, interactive)
	if err != nil {
		fmt.Printf("❌ Error starting TUI chat: %v\n", err)
		fmt.Println("🔄 Falling back to basic chat mode...")
		startBasicChat(mcpServerID, providerID, interactive)
	}
}

func showChatHelp() {
	fmt.Println("\n📖 Chat Commands:")
	fmt.Println("  help, h     - Show this help message")
	fmt.Println("  exit, quit  - End the chat session")
	fmt.Println("  clear, cls  - Clear the screen")
	fmt.Println("  \\n          - Insert line break in message")
	fmt.Println("\n💡 Tips:")
	fmt.Println("  - Use the -i flag for interactive tool approval")
	fmt.Println("  - Specify --provider or --mcp-server for specific resources")
	fmt.Println("  - Messages support multi-line input with \\n")
}

func clearScreen() {
	fmt.Print("\033[2J\033[H") // ANSI escape codes to clear screen
}

// checkStreamingSupport checks if the provider supports streaming
func checkStreamingSupport(provider *types.LLMProvider) bool {
	// For now, we'll check based on provider type
	// In a full implementation, we'd check capabilities from the client
	switch provider.Type {
	case "openai", "anthropic", "local":
		return true
	default:
		return false
	}
}

// processWithStreaming handles streaming response from LLM
func processWithStreaming(ag *agent.Agent, session *types.ConversationSession, message string, display ui.ToolDisplayInterface) error {
	// Use the new Agent streaming method that includes tool support
	streamCh, err := ag.ProcessConversationWithStreaming(session, message, display)
	if err != nil {
		return fmt.Errorf("failed to start streaming: %v", err)
	}
	
	// Clear processing message and start streaming output
	fmt.Print("\r\033[K")  // Clear current line
	fmt.Print("🤖 Agent: ")
	
	var fullResponse strings.Builder
	for chunk := range streamCh {
		if chunk.Error != "" {
			fmt.Printf("\n❌ Error: %s\n", chunk.Error)
			return fmt.Errorf("streaming error: %s", chunk.Error)
		}
		
		if chunk.Content != "" {
			fmt.Print(chunk.Content)
			fullResponse.WriteString(chunk.Content)
		}
		
		if chunk.Done {
			fmt.Println() // New line after response
			break
		}
	}
	
	return nil
}

func init() {
	rootCmd.AddCommand(chatCmd)
	
	chatCmd.Flags().String("mcp-server", "", "MCP server ID to use")
	chatCmd.Flags().String("provider", "", "LLM provider ID to use")
	chatCmd.Flags().BoolP("interactive", "i", false, "Enable interactive mode for tool execution approval")
	chatCmd.Flags().Bool("tui", false, "Use Terminal UI mode (requires bubbletea)")
}