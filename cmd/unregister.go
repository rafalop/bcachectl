package cmd

import (
	"github.com/rafalop/bcachectl/pkg/bcache"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var unregisterCmd = &cobra.Command{
	Use:   "unregister {bcacheX} {bcacheY} ... {deviceN}",
	Short: "unregister formatted bcache device(s)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if IsAdmin {
			all, err := bcache.AllDevs()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			var overallErr error
			for _, dev := range args[0:] {
				err := all.Unregister(dev)
				if err != nil {
					fmt.Println(err)
					overallErr = err
				} else {
					if x, y := all.IsBDevice(dev); x {
						fmt.Println(y.BackingDev, "("+y.ShortName+") was unregistered and but is still formatted.")
					} else if x, z := all.IsCDevice(dev); x {
						fmt.Println(z.Dev, "(cache dev with uuid "+z.UUID+") was unregistered but is still formatted.")
					}
				}
			}
			if overallErr != nil {
				os.Exit(1)
			}
		}
	},
}
