package cli

import (
	command "github.com/Owoade/infracon/cli/command"
	"github.com/spf13/cobra"
)

func main() {

	var rootCommand = &cobra.Command{
		Use:   "infracon",
		Short: "Infracon CLI is a tool for managing infrastructure",
		Run: func(cmd *cobra.Command, args []string) {
			action := args[0]
			if action == "init" {
				command.
			}
		},
	}
}
