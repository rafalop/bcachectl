package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show {bcacheN}",
	Short: "Show detailed information about a bcache device",
	Long:  "If a cache or backing device is supplied, info will be displayed for the bcache device which it is a member of",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		all := allDevs()
		all.RunShow(Format, args[0])
	},
}

func (b *bcache_devs) RunShow(format string, device string) {
	if device == "" {
		fmt.Println("I need a device to show! specify one eg.\n bcachectl show bcache0\n bcachectl show /dev/sda")
		return
	}
	found := false
	if x, y := b.IsBDevice(device); x {
		y.PrintFullInfo(format)
		found = true
	}
	if found == false {
		fmt.Println("Device '" + device + "' is not a registered bcache device")
		return
	}
	return
}
