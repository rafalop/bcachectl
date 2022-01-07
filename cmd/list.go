package cmd
import (

  "github.com/spf13/cobra"
  "fmt"
  "encoding/json"
  "strings"
)

var listCmd = &cobra.Command{
  Use:   "list",
  Short: "list all bcache devices",
  Long: `list all bcache devices along with some info about them. 

possible columns to output with -e:
sequential_cutoff,dirty_data,cache_hit_ratio,cache_hits,cache_misses,writeback_percent`,
  Run: func(cmd *cobra.Command, args []string) {
    all := allDevs()
    all.RunList(Format, Extra)
  },
}

func (b *bcache_devs) RunList(format string, extra string) {
  extra_vals := strings.Split(extra, `,`)
  for _,j := range b.bdevs {
    j.extendMap(extra_vals)
  }
  if format == "json" {
    out := `{`
    jsonb_out, _ := json.Marshal(b.bdevs)
    jsonc_out, _ := json.Marshal(b.cdevs)
    out = out+`"bcache_devs":`+string(jsonb_out)+`, "cache_devs":`+string(jsonc_out)+`}`
    fmt.Println(out)
  } else if format == "short" {
    for _, bdev := range b.bdevs {
      fmt.Printf("%s ", bdev.ShortName)
    }
    fmt.Printf("\n")
  } else {
      b.printTable(extra_vals)
  }
  return
}

