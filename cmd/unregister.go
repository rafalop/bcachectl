package cmd

import (
	"github.com/spf13/cobra"
	"bcachectl/pkg/bcache"
)

//var U *user.User
var unregisterCmd = &cobra.Command{
	Use:   "unregister {bcacheX} {bcacheY} ... {deviceN}",
	Short: "unregister formatted bcache device(s)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if IsAdmin {
			all := bcache.AllDevs()
			all.RunUnregister(args[0:])
		}
	},
}

