package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourusername/syseng-agent/internal/agent"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "AI agent commands",
	Long:  "Commands for interacting with the AI agent",
}

var agentQueryCmd = &cobra.Command{
	Use:   "query [message]",
	Short: "Send a query to the AI agent",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mcpServerID, _ := cmd.Flags().GetString("mcp-server")
		providerID, _ := cmd.Flags().GetString("provider")
		interactive, _ := cmd.Flags().GetBool("interactive")
		
		ag := agent.New(mcpManager, llmManager)
		
		response, err := ag.ProcessRequestWithUI(args[0], mcpServerID, providerID, interactive)
		if err != nil {
			fmt.Printf("Error processing request: %v\n", err)
			return
		}
		
		fmt.Printf("Agent Response: %s\n", response.Message)
		
		if response.Error != "" {
			fmt.Printf("Error: %s\n", response.Error)
		}
	},
}

var agentServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the agent server",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		
		ag := agent.New(mcpManager, llmManager)
		
		fmt.Printf("Starting agent server on port %s...\n", port)
		if err := ag.StartServer(port); err != nil {
			fmt.Printf("Error starting server: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentQueryCmd)
	agentCmd.AddCommand(agentServeCmd)
	
	agentQueryCmd.Flags().String("mcp-server", "", "MCP server ID to use")
	agentQueryCmd.Flags().String("provider", "", "LLM provider ID to use")
	agentQueryCmd.Flags().BoolP("interactive", "i", false, "Enable interactive mode for tool execution approval")
	
	agentServeCmd.Flags().String("port", "8080", "Port to serve on")
}