package cmd

import (
	"bcachectl/pkg/bcache"
	"github.com/spf13/cobra"
)

var detachCmd = &cobra.Command{
	Use:   "detach {cache device} {backing device}",
	Short: "Detaches cache (device) from a backing device",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		all := bcache.AllDevs()
		all.RunDetach(args[0], args[1])
	},
}
