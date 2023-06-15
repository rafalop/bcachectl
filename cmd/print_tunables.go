package cmd

import (
	"bcachectl/pkg/bcache"
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"os"
)

var printTunablesCmd = &cobra.Command{
	Use:   "print-tunables",
	Short: "print existing listable bcache device tunables in yaml format for generating a config file",
	Run: func(cmd *cobra.Command, args []string) {
		all, err := bcache.AllDevs()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		PrintTunables(all)
	},
}

func PrintTunables(b *bcache.BcacheDevs) {
	output := b.GetTunables()
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
