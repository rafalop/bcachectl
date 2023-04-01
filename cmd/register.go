package cmd

import (
	"bcachectl/pkg/bcache"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

// var U *user.User
var registerCmd = &cobra.Command{
	Use:   "register {bcacheX} {bcacheY} ... {deviceN}",
	Short: "register formatted bcache device(s)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if IsAdmin {
			var overallErr error
			var all bcache.BcacheDevs
			for _, dev := range args[0:] {
				err := bcache.Register(dev)
				if err != nil {
					fmt.Println(err)
					overallErr = err
				} else {
					all = bcache.AllDevs()
					if x, y := all.IsBDevice(dev); x {
						fmt.Println(dev, "was registered as", y.ShortName, "and is available for use.")
					} else if x, z := all.IsCDevice(dev); x {
						fmt.Println(dev, "was registered as cache dev with uuid", z.UUID, "and is available for use.")
					}
				}
			}
			if overallErr != nil {
				os.Exit(1)
			}
		}
	},
}
