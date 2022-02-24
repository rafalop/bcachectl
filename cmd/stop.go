package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"unicode"
)

var stopCmd = &cobra.Command{
	Use:   "stop {device}",
	Short: "Try to forcefully stop bcache on a device (remove from sys fs tree)",
	Long:  "Try to forcefully stop bcache on a device. Note, this command only accepts physical devs (not /dev/bcacheX) devices.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if IsAdmin {
			all := allDevs()
			all.RunStop(args[0])
		}
	},
}

func (b *bcache_devs) RunStop(device string) {
	var write_path string
	sn := strings.Split(device, "/")
	shortName := sn[len(sn)-1]
	regexpString := `[0-9]+`
	matched, err := regexp.Match(regexpString, []byte(shortName))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if matched {
		topDev := strings.TrimRightFunc(shortName, func(r rune) bool {
			return unicode.IsNumber(r)
		})
		write_path = SYSFS_BLOCK_ROOT + topDev + `/` + shortName
	} else {
		write_path = SYSFS_BLOCK_ROOT + shortName
	}
	write_path = write_path + `/bcache/`
	if x, _ := b.IsCDevice(device); x {
		write_path = write_path + `set/`
	}
	write_path += `stop`

	err = ioutil.WriteFile(write_path, []byte{1}, 0)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(device, "was stopped, but is still formatted.")
	return
}
