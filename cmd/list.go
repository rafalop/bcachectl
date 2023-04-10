package cmd

import (
	"bcachectl/pkg/bcache"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strings"
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
		all, err := bcache.AllDevs()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		listDevs(all, Format, Extra)
	},
}

func listDevs(b *bcache.BcacheDevs, format string, extra string) {
	var extra_vals []string
	if extra != "" {
		extra_vals = strings.Split(extra, `,`)
	}
	if format == "json" {
		out := `{`
		jsonb_out, _ := json.Marshal(b.Bdevs)
		jsonc_out, _ := json.Marshal(b.Cdevs)
		out = out + `"BcacheDevs":` + string(jsonb_out) + `, "CacheDevs":` + string(jsonc_out) + `}`
		fmt.Println(out)
	} else if format == "short" {
		for _, bdev := range b.Bdevs {
			fmt.Println(bdev.ShortName)
		}
	} else {
		printTable(b, extra_vals)
	}
	return
}

func printTable(b *bcache.BcacheDevs, extra_vals []string) {
	if len(b.Bdevs) > 0 {
		columns := []string{"BcacheDev", "BackingDev", "CacheDev", "cache_mode", "state"}
		for _, val := range extra_vals {
			columns = append(columns, val)
		}
		for _, j := range columns {
			printColumn(j)
		}
		fmt.Printf("\n")
		for _, bdev := range b.Bdevs {
			// we just add these to params map, for ease of printing
			bdev.Parameters["BcacheDev"] = bdev.BcacheDev
			bdev.Parameters["BackingDev"] = bdev.BackingDev
			bdev.Parameters["CacheDev"] = bdev.CacheDev
			for _, j := range columns {
				if bdev.Parameters[j] != nil {
					printColumn(bdev.Parameters[j].(string))
				}
			}
			fmt.Printf("\n")
		}
	} else {
		fmt.Println("None found.")
	}
	fmt.Printf("\n")
	fmt.Println("Registered cache devices:")
	if len(b.Cdevs) > 0 {
		for _, cdev := range b.Cdevs {
			fmt.Println(cdev.Dev, cdev.UUID)
		}
		fmt.Printf("\n")
	} else {
		fmt.Println("None found.")
	}
}

func printColumn(val string) {
	fmt.Printf("%-18s", val)
}
