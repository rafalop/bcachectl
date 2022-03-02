package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"regexp"
	"strings"
	"unicode"
)

var stopCmd = &cobra.Command{
	Use:   "stop {device}",
	Short: "Try to forcefully stop bcache on a device (remove from sys fs tree)",
	Long:  "Try to forcefully stop bcache on a device. Note, this command only accepts physical devs (not /dev/bcacheX) devices.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if IsAdmin {
			all := allDevs()
			return all.RunStop(args[0])
		}
		return nil
	},
}

func (b *bcache_devs) RunStop(device string) error {
	var write_path string
	sn := strings.Split(device, "/")
	shortName := sn[len(sn)-1]
	regexpString := `[0-9]+`
	matched, err := regexp.Match(regexpString, []byte(shortName))
	if err != nil {
		fmt.Println(err)
		return err
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
		return err
	}
	fmt.Println(device, "was stopped, but is still formatted.")
	return nil
}
