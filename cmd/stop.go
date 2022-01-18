package cmd
import (

  "github.com/spf13/cobra"
  "fmt"
  "io/ioutil"
  "os"
  "strings"
  "regexp"
  "unicode"
)

var stopCmd = &cobra.Command{
  Use:   "stop {device}",
  Short: "Try to forcefully stop bcache on a device (remove from sys fs tree)",
  Long: "Try to forcefully stop bcache on a device. This is useful if system is returning messages about the device being busy when you are working with it.",
  Args: cobra.ExactArgs(1),
  Run: func(cmd *cobra.Command, args []string) {
    if IsAdmin {
      RunStop(args[0])
    }
  },
}

func RunStop(device string){
  var write_path string
  sn := strings.Split(device, "/")
  shortName := sn[len(sn)-1]
  regexpString := `[0-9]+`
  matched, err := regexp.Match(regexpString, []byte(shortName))
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  if matched {
    topDev := strings.TrimRightFunc(shortName, func(r rune) bool {
      return unicode.IsNumber(r)
    })
    write_path = SYSFS_BLOCK_ROOT+topDev+`/`+shortName+`/bcache/stop`
  } else {
    write_path = SYSFS_BLOCK_ROOT+shortName+`/bcache/stop`
  }

  err = ioutil.WriteFile(write_path, []byte{1}, 0)
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  fmt.Println(device, "was stopped, but is still formatted.")
  return
}
