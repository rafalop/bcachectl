package cmd
import (

  "github.com/spf13/cobra"
  "gopkg.in/yaml.v2"
  "os"
  "fmt"
)

var printTunablesCmd = &cobra.Command{
  Use:   "print-tunables",
  Short: "print existing listable bcache device tunables in yaml format for generating a config file",
  Run: func(cmd *cobra.Command, args []string) {
    all := allDevs()
    all.PrintTunables()
  },
}

var output = make(map[string]driveConfig)

func (b* bcache_devs) PrintTunables() {
  for _, bdev := range b.bdevs {
    output[bdev.CUUID] = make(driveConfig)
    for _,tunable := range ALLOWED_TUNABLES {
      output[bdev.CUUID][tunable] = bdev.Val(tunable)
    }
  }
  if OutConfigFile != "" {
    out_yaml, _ := yaml.Marshal(&output)
    err := os.WriteFile(OutConfigFile, out_yaml, 0)
    if err != nil {
      fmt.Println(err)
      os.Exit(1)
    }
    fmt.Println("Wrote configuration to", OutConfigFile)
  } else {
    e := yaml.NewEncoder(os.Stdout)
    e.Encode(&output)
  }
}
