package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	newCmd.PersistentFlags().BoolP("debug", "d", false, "If you want to see the debug logs.")
	RootCmd.AddCommand(newCmd)
}

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:     "new service",
	Aliases: []string{"n"},
	Short:   "Some useful generators",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
