package cmd

import (
  "github.com/spf13/cobra"
  "fmt"
  "os"
)
var rootCmd = &cobra.Command{
  Use:   "bcachectl",
  Short: "Simplified administration of bcache devices",
//  Long: `A Fast and Flexible Static Site Generator built with
//                love by spf13 and friends in Go.
//                Complete documentation is available at http://hugo.spf13.com`,
//  Run: func(cmd *cobra.Command, args []string) {
//    // Do Stuff Here
//  },
}

func init() {
  //rootCmd.PersistentFlags().BoolVarP(&OutputFormat, "format", "f", "table", "output format [table|standard|json|short]")

  rootCmd.AddCommand(listCmd)
  var Format string
  listCmd.Flags().StringVarP(&Format, "format", "f", "standard", "Output format [standard|json|short]")
}

func Execute() {
  if err := rootCmd.Execute(); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }
}
