package cmd
import (

  "github.com/spf13/cobra"
  "fmt"
  "io/ioutil"
)

var attachCmd = &cobra.Command{
  Use:   "attach {cache device} {backing device}",
  Short: "Attach an already formatted bcache cache device to a backing device",
  Long: "Attaches a device that has already been formatted as a cache device (exists in sysfs and has uuid) to an already formatted backing device.",
  Args: cobra.ExactArgs(2),
  Run: func(cmd *cobra.Command, args []string) {
    all := allDevs()
    all.RunAttach(args[0], args[1])
  },
}


func (b *bcache_devs) RunAttach(cdev string, bdev string) {
  var x bool
  var y bcache_bdev
  var z bcache_cdev
  if x, y = b.IsBDevice(bdev); ! x {
    fmt.Println(bdev, "does not appear to be a formatted and registered BACKING device.")
    return
  }
  if x, z = b.IsCDevice(cdev); ! x {
    fmt.Println(cdev, "does not appear to be a formatted and registered CACHE device.")
    return
  }
  write_path := SYSFS_BLOCK_ROOT+y.ShortName+`/bcache/attach`
  ioutil.WriteFile(write_path, []byte(z.UUID), 0)
  y.FindCUUID()
  if y.CUUID != z.UUID {
    fmt.Println("Cache device could not be attached. Is there already a cache set associated with the device?\n")
    return
  }
  fmt.Println("Cache device", cdev, "was attached as cache for", bdev)
}
