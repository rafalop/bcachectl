package cmd
import (

  "github.com/spf13/cobra"
  "fmt"
  "io/ioutil"
)

var registerCmd = &cobra.Command{
  Use:   "register {device1} {device2} ... {deviceN}",
  Short: "register formatted bcache device(s)",
  Args: cobra.MinimumNArgs(1),
  Run: func(cmd *cobra.Command, args []string) {
    if IsAdmin {
      RunRegister(args[0:])
    }
  },
}

func RunRegister(devices []string){
  var write_path string
  write_path = SYSFS_BCACHE_ROOT+`register`
  all := allDevs()
  for _, device := range devices {
    if x := all.IsBCDevice(device); x {
      fmt.Println("Device is already registered.")
    } else {
      ioutil.WriteFile(write_path, []byte(device), 0)
      all = allDevs()
      done := false
      if x, _ := all.IsCDevice(device); x {
        done = true
      }
      if done {
        fmt.Println(device, "was registered.")
        all.printTable()
      } else {
        fmt.Println("Couldn't register "+device+". If it is a backing device with cache device attached, you should try to register the cache device instead.")
      }
    }
  }
  fmt.Println()
  all.printTable()
  return
}
