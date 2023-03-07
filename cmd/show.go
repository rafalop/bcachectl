package cmd

import (
	"github.com/spf13/cobra"
	"bcachectl/pkg/bcache"
)

var showCmd = &cobra.Command{
	Use:   "show {bcacheN}",
	Short: "Show detailed information about a bcache device",
	Long:  "If a cache or backing device is supplied, info will be displayed for the bcache device which it is a member of",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		all := bcache.AllDevs()
		all.RunShow(Format, args[0])
	},
}

