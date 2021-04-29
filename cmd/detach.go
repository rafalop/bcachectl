package cmd
import (

  "github.com/spf13/cobra"
  "fmt"
  "io/ioutil"
)

var detachCmd = &cobra.Command{
  Use:   "detach {cache device} {backing device}",
  Short: "Detaches cache (device) from a backing device",
  Args: cobra.ExactArgs(2),
  Run: func(cmd *cobra.Command, args []string) {
    all := allDevs()
    all.RunDetach(args[0], args[1])
  },
}

func (b *bcache_devs) RunDetach (cdev string, bdev string) {
  var writepath string = SYSFS_BLOCK_ROOT
  var x bool
  var y bcache_cdev
  var z bcache_bdev
  x, y = b.IsCDevice(cdev)
  if ! x {
    fmt.Println(cdev, "is not a registered cache device.")
    return
  }
  x, z = b.IsBDevice(bdev)
  if ! x {
    fmt.Println(bdev, "is not a registered backing device.")
    return
  }
  writepath = writepath+z.ShortName+`/bcache/detach`
  ioutil.WriteFile(writepath, []byte(y.UUID), 0)
  fmt.Println("Detached cache dev", cdev, "from "+bdev)
}
