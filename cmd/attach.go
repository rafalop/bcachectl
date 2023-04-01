package cmd

import (
	"bcachectl/pkg/bcache"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var attachCmd = &cobra.Command{
	Use:   "attach {cache device} {backing device}",
	Short: "Attach an already formatted bcache cache device to a backing device",
	Long:  "Attaches a device that has already been formatted as a cache device (exists in sysfs and has uuid) to an already formatted backing device.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		all := bcache.AllDevs()
		err := all.Attach(args[0], args[1])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			fmt.Println("Cache device", args[1], "was attached as cache for", args[0]+".")
		}
	},
}
