package cmd

import (
	"github.com/spf13/cobra"
	"bcachectl/pkg/bcache"
	"fmt"
)

var superCmd = &cobra.Command{
	Use:   "super {device}",
	Short: "Print bcache superblock of a system device",
	Long:  "Print the superblock, a wrapper for `bcache-super-show`. The device provided should be a system device, not a bcache (bcacheX) device.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		//allDevs()
		fmt.Println(bcache.GetSuperBlock(args[0]))
	},
}
