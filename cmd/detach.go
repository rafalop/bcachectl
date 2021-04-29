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
  x, y = b.IsCDevice(cdev)
  if ! x {
    fmt.Println(cdev, "is not a registered cache device.")
    return
  }
  for _, dev := range b.bdevs {
    if dev.BackingDev == bdev ||
      dev.ShortName == bdev ||
      dev.BcacheDev == bdev {
      if dev.CacheDev == "(none attached)" {
        fmt.Println("No cache device currently attached to", bdev)
        return
      }
      writepath = writepath+dev.ShortName+`/bcache/detach`
      break
    }
  }
  ioutil.WriteFile(writepath, []byte(y.UUID), 0)
  fmt.Println("Detached cache dev", cdev, "from "+bdev)
}
