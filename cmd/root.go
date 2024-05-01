package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const version = "v1.0.0"

func init() {
	RootCmd.PersistentFlags().BoolP("debug", "d", false, "If you want to see the debug logs.")
	RootCmd.PersistentFlags().BoolP("force", "f", false, "Force overide existing files without asking.")
	RootCmd.PersistentFlags().StringP("folder", "b", "", "If you want to specify the base folder of the project.")

	viper.BindPFlag("folder", RootCmd.PersistentFlags().Lookup("folder"))
	viper.BindPFlag("force", RootCmd.PersistentFlags().Lookup("force"))
	viper.BindPFlag("debug", RootCmd.PersistentFlags().Lookup("debug"))
}

// RootCmd is the root command of kit
var RootCmd = &cobra.Command{
	Use:     "go-kit",
	Short:   "Go-Kit CLI",
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute runs the root command
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}
