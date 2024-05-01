package cmd

import (
	"go-kit-demo/cmd/gen"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var newServiceCmd = &cobra.Command{
	Use:     "service name",
	Short:   "Generate new service",
	Aliases: []string{"s"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			logrus.Error("You must provide a name for the service")
			cmd.Help()
			return
		}
		g := gen.NewNewService(args[0])
		if err := g.Generate(); err != nil {
			logrus.Error(err)
		}
	},
}

func init() {
	newCmd.AddCommand(newServiceCmd)
	newServiceCmd.Flags().StringP("module", "m", "", "The module name that you plan to set in the project")
	viper.BindPFlag("module", newServiceCmd.Flags().Lookup("module"))
}
