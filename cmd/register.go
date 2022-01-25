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
    //fmt.Println("write_path:", write_path, "device:", device)
    if x := all.IsBCDevice(device); x {
      fmt.Println(device, "is already registered.")
    } else {
      err := ioutil.WriteFile(write_path, []byte(device), 0)
      all = allDevs()
      if x, y := all.IsBDevice(device); x {
        fmt.Println(device, "was registered as", y.ShortName+".")
      } else if x, y := all.IsCDevice(device); x {
        fmt.Println(device, "was registered as a cache device with uuid", y.UUID+".")
      } else {
        fmt.Println("Couldn't register "+device+". Errors:", err)
      }
    }
  }
  fmt.Println()
  return
}
