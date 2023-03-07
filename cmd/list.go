package cmd

import (
	"bcachectl/pkg/bcache"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list all bcache devices",
	Long: `list all bcache devices along with some info about them. 

possible columns to output with -e:
sequential_cutoff
dirty_data
cache_hit_ratio
cache_hits
cache_misses
writeback_percent`,
	Run: func(cmd *cobra.Command, args []string) {
		all := bcache.AllDevs()
		all.RunList(Format, Extra)
	},
}

