package cmd

import (
	"bcachectl/pkg/bcache"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var detachCmd = &cobra.Command{
	Use:   "detach {cache device} {backing device}",
	Short: "Detaches cache (device) from a backing device",
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
		if b.CacheDev == bcache.NONE_ATTACHED {
			fmt.Println("device " + args[1] + " has no cache attached, nothing to do.")
		} else {
			err := all.Detach(args[0], args[1])
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			} else {
				fmt.Println("Detached cache dev", args[0], "from", b.BackingDev+" ("+b.ShortName+")")
			}
		}
	},
}
