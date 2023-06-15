package cmd

import (
	"github.com/rafalop/bcachectl/pkg/bcache"
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
		all, err := bcache.AllDevs()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		x, b := all.IsBDevice(args[1])
		if !x {
			fmt.Println(args[1] + " is not a bcache device.")
			os.Exit(1)
		}
		if b.CacheDev != bcache.NONE_ATTACHED {
			fmt.Println(args[1] + " (" + b.ShortName + ") already has cache attached (" + b.CacheDev + ")")
		} else {
			err := all.Attach(args[0], args[1])
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			} else {
				fmt.Println("Cache device", args[0], "was attached as cache for", b.BackingDev+" ("+b.ShortName+")")
			}
		}
	},
}
