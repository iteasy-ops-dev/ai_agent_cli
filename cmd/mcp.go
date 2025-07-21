package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/iteasy-ops-dev/syseng-agent/internal/mcp"
	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
	"github.com/spf13/cobra"
)

var mcpManager *mcp.Manager

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP servers",
	Long:  "Commands for managing Model Context Protocol (MCP) servers",
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all MCP servers",
	Run: func(cmd *cobra.Command, args []string) {
		servers := mcpManager.ListServers()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tURL\tTRANSPORT\tSTATUS\tLAST_PING")

		for _, server := range servers {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				server.ID,
				server.Name,
				server.URL,
				server.Transport,
				server.Status,
				server.LastPing.Format("15:04:05"),
			)
		}

		w.Flush()
	},
}

var mcpAddCmd = &cobra.Command{
	Use:   "add [name] [url] [transport]",
	Short: "Add a new MCP server",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		server := &types.MCPServer{
			Name:      args[0],
			URL:       args[1],
			Transport: args[2],
		}

		if err := mcpManager.AddServer(server); err != nil {
			fmt.Printf("Error adding server: %v\n", err)
			return
		}

		fmt.Printf("Server %s added successfully with ID: %s\n", server.Name, server.ID)
	},
}

var mcpRemoveCmd = &cobra.Command{
	Use:   "remove [id]",
	Short: "Remove an MCP server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := mcpManager.RemoveServer(args[0]); err != nil {
			fmt.Printf("Error removing server: %v\n", err)
			return
		}

		fmt.Printf("Server %s removed successfully\n", args[0])
	},
}

var mcpShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show details of an MCP server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		server, err := mcpManager.GetServer(args[0])
		if err != nil {
			fmt.Printf("Error getting server: %v\n", err)
			return
		}

		data, err := json.MarshalIndent(server, "", "  ")
		if err != nil {
			fmt.Printf("Error formatting server data: %v\n", err)
			return
		}

		fmt.Println(string(data))
	},
}

var mcpToolsCmd = &cobra.Command{
	Use:   "tools [server-id]",
	Short: "List available tools for an MCP server",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// Show all tools from all servers
			allTools := mcpManager.GetAllTools()

			for serverName, tools := range allTools {
				fmt.Printf("\n=== %s ===\n", serverName)
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "TOOL\tDESCRIPTION")

				for _, tool := range tools {
					fmt.Fprintf(w, "%s\t%s\n", tool.Name, tool.Description)
				}

				w.Flush()
			}
		} else {
			// Show tools for specific server
			tools, err := mcpManager.GetServerTools(args[0])
			if err != nil {
				fmt.Printf("Error getting tools: %v\n", err)
				return
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TOOL\tDESCRIPTION")

			for _, tool := range tools {
				fmt.Fprintf(w, "%s\t%s\n", tool.Name, tool.Description)
			}

			w.Flush()
		}
	},
}

var mcpCallCmd = &cobra.Command{
	Use:   "call [server-id] [tool-name] [arguments-json]",
	Short: "Call a tool on an MCP server",
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		serverID := args[0]
		toolName := args[1]

		var arguments map[string]interface{}
		if len(args) > 2 {
			if err := json.Unmarshal([]byte(args[2]), &arguments); err != nil {
				fmt.Printf("Error parsing arguments JSON: %v\n", err)
				return
			}
		}

		result, err := mcpManager.CallTool(serverID, toolName, arguments)
		if err != nil {
			fmt.Printf("Error calling tool: %v\n", err)
			return
		}

		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Printf("Error formatting result: %v\n", err)
			return
		}

		fmt.Println(string(data))
	},
}

func init() {
	mcpManager = mcp.NewManager()

	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpAddCmd)
	mcpCmd.AddCommand(mcpRemoveCmd)
	mcpCmd.AddCommand(mcpShowCmd)
	mcpCmd.AddCommand(mcpToolsCmd)
	mcpCmd.AddCommand(mcpCallCmd)
}
