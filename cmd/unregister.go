package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
)

//var U *user.User
var unregisterCmd = &cobra.Command{
	Use:   "unregister {bcacheX} {bcacheY} ... {deviceN}",
	Short: "unregister formatted bcache device(s)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if IsAdmin {
			all := allDevs()
			all.RunUnregister(args[0:])
		}
	},
}

func (b *bcache_devs) RunUnregister(devices []string) {
	var write_path string
	for _, device := range devices {
		if x, bdev := b.IsBDevice(device); x {
			write_path = SYSFS_BLOCK_ROOT + bdev.ShortName + `/bcache/stop`
			ioutil.WriteFile(write_path, []byte{1}, 0)
			fmt.Println(device, "(backing device) was unregistered, but is still formatted.")
			return
		}
		if x, cdev := b.IsCDevice(device); x {
			write_path = SYSFS_BCACHE_ROOT + cdev.UUID + `/stop`
			ioutil.WriteFile(write_path, []byte{1}, 0)
			fmt.Println(device, "(cache device) was unregistered, but is still formatted.")
			return
		}
		fmt.Println(device + " does not appear to be a registered bcache device.")
	}
	return
}
