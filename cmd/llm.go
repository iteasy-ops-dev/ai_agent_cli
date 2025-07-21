package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/iteasy-ops-dev/syseng-agent/internal/llm"
	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
	"github.com/spf13/cobra"
)

var llmManager *llm.Manager

var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "Manage LLM providers",
	Long:  "Commands for managing Large Language Model (LLM) providers",
}

var llmListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all LLM providers",
	Run: func(cmd *cobra.Command, args []string) {
		providers := llmManager.ListProviders()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tTYPE\tMODEL\tACTIVE\tCREATED")

		for _, provider := range providers {
			active := "No"
			if provider.IsActive {
				active = "Yes"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				provider.ID,
				provider.Name,
				provider.Type,
				provider.Model,
				active,
				provider.CreatedAt.Format("2006-01-02"),
			)
		}

		w.Flush()
	},
}

var llmAddCmd = &cobra.Command{
	Use:   "add [name] [type] [model]",
	Short: "Add a new LLM provider",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		apiKey, _ := cmd.Flags().GetString("api-key")
		endpoint, _ := cmd.Flags().GetString("endpoint")

		provider := &types.LLMProvider{
			Name:     args[0],
			Type:     args[1],
			Model:    args[2],
			APIKey:   apiKey,
			Endpoint: endpoint,
		}

		if err := llmManager.AddProvider(provider); err != nil {
			fmt.Printf("Error adding provider: %v\n", err)
			return
		}

		fmt.Printf("Provider %s added successfully with ID: %s\n", provider.Name, provider.ID)
	},
}

var llmRemoveCmd = &cobra.Command{
	Use:   "remove [id]",
	Short: "Remove an LLM provider",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := llmManager.RemoveProvider(args[0]); err != nil {
			fmt.Printf("Error removing provider: %v\n", err)
			return
		}

		fmt.Printf("Provider %s removed successfully\n", args[0])
	},
}

var llmShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show details of an LLM provider",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider, err := llmManager.GetProvider(args[0])
		if err != nil {
			fmt.Printf("Error getting provider: %v\n", err)
			return
		}

		providerCopy := *provider
		providerCopy.APIKey = "***"

		data, err := json.MarshalIndent(providerCopy, "", "  ")
		if err != nil {
			fmt.Printf("Error formatting provider data: %v\n", err)
			return
		}

		fmt.Println(string(data))
	},
}

var llmSetActiveCmd = &cobra.Command{
	Use:   "set-active [id]",
	Short: "Set an LLM provider as active",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := llmManager.SetActiveProvider(args[0]); err != nil {
			fmt.Printf("Error setting active provider: %v\n", err)
			return
		}

		fmt.Printf("Provider %s set as active\n", args[0])
	},
}

func init() {
	llmManager = llm.NewManager()

	rootCmd.AddCommand(llmCmd)
	llmCmd.AddCommand(llmListCmd)
	llmCmd.AddCommand(llmAddCmd)
	llmCmd.AddCommand(llmRemoveCmd)
	llmCmd.AddCommand(llmShowCmd)
	llmCmd.AddCommand(llmSetActiveCmd)

	llmAddCmd.Flags().String("api-key", "", "API key for the provider")
	llmAddCmd.Flags().String("endpoint", "", "Endpoint URL for local providers")
}
