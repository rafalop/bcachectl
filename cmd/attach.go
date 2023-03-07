package cmd

import (
	"github.com/spf13/cobra"
	"bcachectl/pkg/bcache"
)

var attachCmd = &cobra.Command{
	Use:   "attach {cache device} {backing device}",
	Short: "Attach an already formatted bcache cache device to a backing device",
	Long:  "Attaches a device that has already been formatted as a cache device (exists in sysfs and has uuid) to an already formatted backing device.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		all := bcache.AllDevs()
		all.RunAttach(args[0], args[1])
	},
}

