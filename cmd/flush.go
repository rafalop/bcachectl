package cmd

import (
	"bcachectl/pkg/bcache"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var flushCmd = &cobra.Command{
	Use:   "flush {bcacheN}|all",
	Short: "Flush devices dirty data from cache",
	Long:  "Flush the dirty data for one or all bcache devices. Only used when cache is in writeback mode.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if IsAdmin {
			err = Flush(args[0])
			if err != nil {
				fmt.Println("Error flushing: " + err.Error())
				os.Exit(1)
			}
		}
	},
}

func Flush(device string) (returnErr error) {
	var x bool
	var y bcache.Bcache_bdev
	b, err := bcache.AllDevs()
	if err != nil {
		return errors.New("Error getting bcache devices:" + err.Error())
	}
	if device == "" {
		return errors.New("no device supplied")
	} else if x, y = b.IsBDevice(device); device != "" && !x {
		return errors.New(device + " is not a valid bcache device")
	} else if device != "all" {
		// Flush single
		e1, e2 := y.FlushCache()
		if e1 != nil {
			returnErr = errors.New("could not flush " + y.ShortName + ": " + e1.Error())
		} else if e2 != nil {
			returnErr = errors.New("could reset writeback settings" + y.ShortName + ": " + e2.Error())
		} else if e1 != nil && e2 != nil {
			returnErr = errors.New("errors during flush: " + y.ShortName + ": " + e1.Error() + ", " + e2.Error())
		} else {
			fmt.Println("cache for " + y.ShortName + " was flushed successfully.")
		}
	} else if device == "all" {
		// Flush all
		c := make(chan string, len(b.Bdevs))
		for _, dev := range b.Bdevs {
			go func(d bcache.Bcache_bdev) {
				var e1, e2 error
				e1, e2 = d.FlushCache()
				if e1 != nil || e2 != nil {
					c <- "error while flushing " + d.ShortName + ": err1:" + e1.Error() + "err2:" + e2.Error() + "\n"
					returnErr = errors.New("couldn't flush one or more devices.")
				} else {
					c <- "cache for " + d.ShortName + " was flushed successfully."
				}
			}(dev)
		}
		for range b.Bdevs {
			fmt.Printf(<-c)
		}
	}
	return
}
