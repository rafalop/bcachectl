package cmd
import (

  "github.com/spf13/cobra"
  "fmt"
  "encoding/json"
)

var listCmd = &cobra.Command{
  Use:   "list",
  Short: "list all bcache devices",
  Run: func(cmd *cobra.Command, args []string) {
    if Format == "json" {
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
        b.printTable()
    }
    return
  },
}

