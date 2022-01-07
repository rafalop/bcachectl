package cmd
import (

  "github.com/spf13/cobra"
  "fmt"
  //"errors"
  "time"
)

var flushCmd = &cobra.Command{
  Use:   "flush {bcacheN}",
  Short: "Flush devices dirty data from cache",
  Long: "Flush the dirty data for one or all bcache devices. Only used when cache is in writeback mode.",
  Args: cobra.MinimumNArgs(0),
  Run: func(cmd *cobra.Command, args []string) {
    if IsAdmin {
      all := allDevs()
      if ApplyToAll {
        all.RunFlush("", true)
      } else {
        all.RunFlush(args[0], false)
      }
    }
  },
}

func (b *bcache_devs) RunFlush(device string, all bool) {
  var x bool
  var y bcache_bdev
  var err error
  if device == "" && ! all {
    fmt.Println("I need a device to flush, eg.\n bcachectl flush bcache0\n")
    return
  } else if x, y = b.IsBDevice(device); device != "" && ! x {
    fmt.Println(device, "does not appear to be a valid bcache device (expecting valid bcacheXY)\n")
    return
  } else if ! all {
    // Flush single
    y.FlushCache()
  } else {
    // Flush all
    c := make(chan string, len(b.bdevs))
    for _, dev := range b.bdevs {
      go func (d bcache_bdev){
        d.FlushCache()
        if err != nil {
          c <- "error flushing "+d.ShortName+": "+err.Error()+"\n"
        } else {
          c <- ""
        }
      }(dev)
    }
    for range b.bdevs {
      fmt.Printf(<-c)
    }
  }
  return
}

func (b *bcache_bdev) FlushCache() {
  var err error
  // First check if current mode is writeback
  r := b.Val(`cache_mode`)
  if r != "writeback" {
    fmt.Println(b.ShortName, "is not using writeback mode, nothing to flush")
    return
  }

  // Set writeback_delay to something short
  write_delay := b.Val(`writeback_delay`)
  err = b.ChangeTunable(`writeback_delay`, `1`)
  if err != nil {
    fmt.Println("error setting writeback_delay:", err)
    return
  }

  // To achieve flush, we set cachemode to writethrough until state is clean
  err = b.ChangeTunable(`cache_mode`, `writethrough`)
  if err != nil {
    fmt.Println("error setting cache_mode:", err)
    return
  }
  tries := 0
  for {
    if tries == 30 {
      fmt.Println("could not complete flush after within 30 seconds. you could try manually setting cache mode to `writethrough` and wait longer for it to flush.\n")
      break
    }
    state := b.Val(`state`)
    if state == `clean` {
      break
    } else {
      time.Sleep(1*time.Second)
    }
    tries += 1
  }
  fmt.Println("cache was flushed successfully for", b.ShortName, "(device reached clean state)")

  // If we entered this function, cache mode must have been writeback, we change back
  err = b.ChangeTunable(`cache_mode`, `writeback`)
  if err != nil {
    fmt.Println("unable to set mode back to writeback:", err)
  }
  // Set original writeback delay
  err = b.ChangeTunable(`writeback_delay`, write_delay)
  if err != nil {
    fmt.Println("unable to set writeback_delay back to original value!\n")
  }
  return
}
