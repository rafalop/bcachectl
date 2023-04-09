package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"bcachectl/pkg/bcache"
	//"errors"
	"os"
	//"time"
)

var flushCmd = &cobra.Command{
	Use:   "flush {bcacheN}",
	Short: "Flush devices dirty data from cache",
	Long:  "Flush the dirty data for one or all bcache devices. Only used when cache is in writeback mode.",
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if IsAdmin {
			all, err := bcache.AllDevs()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if ApplyToAll {
				Flush(all, "", true)
			} else if len(args) == 0 {
				Flush(all, "", false)
			} else {
				Flush(all, args[0], false)
			}
		}
	},
}

func Flush(b *bcache.BcacheDevs, device string, all bool){
	var x bool
	var y bcache.Bcache_bdev
	var err error
	var successMsg = fmt.Sprintf("device %s flushed successfully\n", device)
	if device == "" && !all {
		fmt.Println("I need a device to flush, eg.\n bcachectl flush bcache0\n\nor use -a to flush all.")
		//return errors.New("no device supplied")
	} else if x, y = b.IsBDevice(device); device != "" && !x {
		fmt.Println(device, "does not appear to be a valid bcache device (expecting valid bcacheXY)\n")
		//return errors.New(device + " is not a valid bcache device")
	} else if !all {
		// Flush single
		y.FlushCache()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			fmt.Printf(successMsg)
		}
	} else {
		// Flush all
		c := make(chan error, len(b.Bdevs))
		for _, dev := range b.Bdevs {
			go func(d bcache.Bcache_bdev) {
				c <- d.FlushCache()
			}(dev)
		}
		for range b.Bdevs {
			err = <-c
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Printf(successMsg)
			}
		}
	}
	return
}
