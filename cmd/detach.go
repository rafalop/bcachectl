package cmd
import (

  "github.com/spf13/cobra"
  "fmt"
  "io/ioutil"
)

var detachCmd = &cobra.Command{
  Use:   "detach {device}",
  Short: "Detaches cache from a bcache cache device.",
  Args: cobra.ExactArgs(1),
  Run: func(cmd *cobra.Command, args []string) {
    all := allDevs()
    all.RunDetach(args[0])
  },
}

func (b *bcache_devs) RunDetach (dev string) {
  var writepath string = SYSFS_BLOCK_ROOT
  var printdev bcache_bdev
  if ! b.IsBCDevice(dev) {
    fmt.Println(dev, "is not a bcache registered device.")
    return
  }
  for _, bdev := range b.bdevs {
    if bdev.BackingDevs[0] == dev ||
      bdev.CacheDevs[0] == dev ||
      bdev.ShortName == dev ||
      bdev.BcacheDev == dev {
      if bdev.CacheDevs[0] == "(none attached)" {
        fmt.Println("No cache device currently attached.")
        return
      }
      writepath = writepath+bdev.ShortName+`/bcache/detach`
      printdev = bdev
      break
    }
  }
  ioutil.WriteFile(writepath, []byte{1}, 0)
  fmt.Println("Detached cache from "+dev)
  printdev.PrintFullInfo("standard")
}
