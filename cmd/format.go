package cmd

import (
	"bcachectl/pkg/bcache"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var formatCmd = &cobra.Command{
	Use:   "format -[B|C] {device1} -[B|C] {device2} ... -[B|C] {deviceN}",
	Short: "format a bcache backing and/or cache device(s)",
	Long:  "Add/Format/Create a bcache device potentially auto attaching a cache device to a backing device if both are specified together (-B) and (-C). This is a wrapper for `make-bcache` and will use the same arguments, eg. -B {backing dev} -C {cache dev}",
	Run: func(cmd *cobra.Command, args []string) {
		if IsAdmin && (NewBDev != "" || NewCDev != "") {
			all, err := bcache.AllDevs()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			err = all.Create(NewBDev, NewCDev, Wipe, WriteBack)
			if err == nil {
				fmt.Println("Completed formatting device(s):", NewBDev, NewCDev)
			} else {
				fmt.Println(err)
				os.Exit(1)
			}
		} else {
			fmt.Println("I need at least one backing dev (-B) or one cache dev (-C) to format!")
		}
	},
}
