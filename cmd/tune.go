package cmd

import (
	"bcachectl/pkg/bcache"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var tuneCmd = &cobra.Command{
	Use:   "tune [{bcacheN|all} {tunable:value}] | [from-file /some/config/file]",
	Short: "Change tunable for a bcache device or tune devices from a config file",
	Long:  "Tune a bcache device.  Using 'from-file /file/name' will read tunables from a config file and tune each specified device or 'all' devices. Allowed tunables are:\n" + bcache.ALLOWED_TUNABLES_DESCRIPTIONS,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if IsAdmin {
			all, err := bcache.AllDevs()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if args[0] == "from-file" {
				err = all.TuneFromFile(args[1])
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				} else {
					fmt.Println("Applied tunables from", args[1])
				}
			} else {
				tune(all, args[0], args[1])
			}
		}
	},
}

func tune(b *bcache.BcacheDevs, device string, tunable string){
	var all bool = false
	var y bcache.Bcache_bdev
	var x bool
	var err error
	// overallErr tracks if any error occurs while tuning all devs
	var overallErr error
	if device == "all" {
		all = true
	} 
	if device == "" { 
		fmt.Println("I need a registered device to tune, eg.\n bcachectl tune bcache0 tunable_name:tunable_val\n\nor use \"all\" to apply the same tunable to all registered devices.")
	} else if !all {
		// Tune single
		if x, y = b.IsBDevice(device); !x {
			fmt.Println(device, "does not appear to be a valid bcache device (expecting valid bcacheXY)\n")
		} else {
			err = y.Tune(tunable)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			} else {
				fmt.Printf("device %s was tuned successfully (%s)\n", device, tunable)
			}
		}
	} else {
		// Tune all
		for _, dev := range b.Bdevs {
			err = dev.Tune(tunable)
			if err != nil {
				fmt.Printf("could not tune %s: %s\n", dev.ShortName, err)
				overallErr = err
			} else {
				fmt.Printf("device %s was tuned successfully (%s)\n", dev.ShortName, tunable)
			}
		}
	}
	if overallErr != nil {
		os.Exit(1)
	}
	return
}
