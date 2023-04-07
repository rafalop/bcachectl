package cmd

import (
	"bcachectl/pkg/bcache"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var tuneCmd = &cobra.Command{
	Use:   "tune [{bcacheN} {tunable:value}] | [from-file /some/config/file]",
	Short: "Change tunable for a bcache device or tune devices from a config file",
	Long:  "Tune bcache by writing to sysfs. Using 'from-file /file/name' will read tunables from a config file and tune each specified device or 'all' devices. Allowed tunables are:\n" + bcache.ALLOWED_TUNABLES_DESCRIPTIONS,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if IsAdmin {
			var errType int
			var err error
			all, err := bcache.AllDevs()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if args[0] == "from-file" {
				err = all.TuneFromFile(args[1])
			} else {
				if args[0] == "" {
					fmt.Println("I need a device to work on, eg.\n bcachectl tune bcache0 cache_mode:writeback\n")
					os.Exit(1)
				}
				err = all.Tune(args[0], args[1])
				if err != nil {
					fmt.Println(err)
					fmt.Println(bcache.ALLOWED_TUNABLES_ERRORSTRING)
					os.Exit(errType)
				}
			}
		}
	},
}
