package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/rafalop/bcachectl/pkg/bcache"
	"github.com/spf13/cobra"
	"os"
)

var showCmd = &cobra.Command{
	Use:   "show {bcacheN}",
	Short: "Show detailed information about a bcache device",
	Long:  "If a cache or backing device is supplied, info will be displayed for the bcache device which it is a member of",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		all, err := bcache.AllDevs()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		show(all, Format, args[0])
	},
}

func show(b *bcache.BcacheDevs, format string, device string) (err error) {
	if device == "" {
		fmt.Println("I need a device to show! specify one eg.\n bcachectl show bcache0\n bcachectl show /dev/sda")
		os.Exit(1)
		return
	}
	found := false
	if x, y := b.IsBDevice(device); x {
		printFullInfo(&y, format)
		found = true
	}
	if found == false {
		fmt.Println("Device '" + device + "' is not a registered bcache device")
		os.Exit(1)
	}
	return
}

func printFullInfo(b *bcache.Bcache_bdev, format string) {
	if format == "json" {
		json_out, _ := json.Marshal(b)
		fmt.Println(string(json_out))
	} else {
		fmt.Printf("%-30s%s\n", "ShortName:", b.ShortName)
		fmt.Printf("%-30s%s\n", "Bcache Dev UUID:", b.BUUID)
		fmt.Printf("%-30s%s\n", "Cache Set UUID:", b.CUUID)
		fmt.Printf("%-30s%s\n", "Backing device:", b.BackingDev)
		fmt.Printf("%-30s%s\n", "Cache device:", b.CacheDev)
		for k, v := range b.Parameters {
			if v != "" {
				fmt.Printf("%-30s%s\n", k+`:`, v)
			} else {
				fmt.Printf("%-30s%s\n", k+`:`, "N\\A")
			}
		}
	}
	return
}
