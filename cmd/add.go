package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"regexp"
)

var addCmd = &cobra.Command{
	Use:   "add -[B|C] {device1} -[B|C] {device2} ... -[B|C] {deviceN}",
	Short: "add (format) bcache backing and/or cache device(s)",
	Long:  "Add/Format/Create one or more bcache devices, potentially auto attaching a cache device to a backing device if both are specified together (-B) and (-C). This is a wrapper for `make-bcache` and will use the same arguments, eg. -B {backing dev} -C {cache dev}",
	Run: func(cmd *cobra.Command, args []string) {
		if IsAdmin && (NewBDev != "" || NewCDev != "") {
			allDevs := allDevs()
			allDevs.RunCreate(NewBDev, NewCDev)
		} else {
			fmt.Println("I need at least one backing dev (-B) or one cache dev (-C) to format!")
		}
	},
}

func (b *bcache_devs) RunCreate(newbdev string, newcdev string) {
	bcache_cmd := `/usr/sbin/make-bcache`
	var out string
	if newcdev != "" {
		bcache_cmd = bcache_cmd + ` -C ` + newcdev
		if Wipe {
			bcache_cmd = bcache_cmd + ` --wipe-bcache`
			b.RunStop(newcdev)
			out, _ = RunSystemCommand(`/sbin/wipefs -a ` + newcdev)
			fmt.Println(out)
		}
	}
	if newbdev != "" {
		bcache_cmd = bcache_cmd + ` -B ` + newbdev
		if Wipe {
			b.RunStop(newbdev)
			out, _ := RunSystemCommand(`/sbin/wipefs -a ` + newbdev)
			fmt.Println(out)
			bcache_cmd = bcache_cmd + ` --wipe-bcache`
		}
	}
	if WriteBack {
		bcache_cmd = bcache_cmd + " --writeback"
	}
	out, err := RunSystemCommand(bcache_cmd)
	if err == nil {
		fmt.Println("Completed formatting device(s):", newbdev, newcdev)
		if newbdev != "" {
			RunRegister([]string{newbdev})
		}
		if newcdev != "" {
			RunRegister([]string{newcdev})
		}
	}
	already_formatted, _ := regexp.MatchString("Already a bcache device", out)
	busy, _ := regexp.MatchString("Device or resource busy", out)
	existing_super, _ := regexp.MatchString("non-bcache superblock", out)
	if busy {
		fmt.Println("Device is busy - is it already registered bcache dev or mounted?")
	}
	if already_formatted || existing_super {
		fmt.Println(out)
		fmt.Printf("If you REALLY want to format this device, make sure it is not registered and use the --wipe-super flag (will erase ANY superblocks and filesystems!):\n  bcachectl unregister {device}\n  bcachectl add -(B|C) {device} --wipe-super\n")
	}
	if err != nil {
		os.Exit(1)
	}
	return
}
