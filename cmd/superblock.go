package cmd

import (
	"fmt"
	"github.com/rafalop/bcachectl/pkg/bcache"
	"github.com/spf13/cobra"
	"os"
)

var superCmd = &cobra.Command{
	Use:   "super {device}",
	Short: "Print bcache superblock of a system device",
	Long:  "Print the superblock, a wrapper for `bcache-super-show`. The device provided should be a system device, not a bcache (bcacheX) device.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		out, err := bcache.GetSuperBlock(args[0])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(out)
	},
}
