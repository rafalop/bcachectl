package cmd
import (

  "github.com/spf13/cobra"
  "fmt"
  "regexp"
  "os"
)

var createCmd = &cobra.Command{
  Use:   "create -[B|C] {device1} -[B|C] {device2} ... -[B|C] {deviceN}",
  Short: "create (format) a bcache device",
  Long: "Create a bcache device, potentially a cached bcache device if both a backing (-B) and cache (-C) device are specified together. This is a wrapper for `make-bcache` and will use the same arguments, eg. -B {backing dev} -C {cache dev}",
  Run: func(cmd *cobra.Command, args []string) {
    if IsAdmin && (NewBDev != "" || NewCDev != ""){
      RunCreate(NewBDev, NewCDev)
    } else {
      fmt.Println("I need at least one backing dev (-B) or one cache dev (-C) to format!")
    }
  },
}

func RunCreate(newbdev string, newcdev string){
  bcache_cmd := `/usr/sbin/make-bcache`
  if newcdev != "" {
    bcache_cmd = bcache_cmd+` -C `+newcdev
  }
  if newbdev != "" {
    bcache_cmd = bcache_cmd+` -B `+newbdev
  }
  if Wipe {
    bcache_cmd = bcache_cmd+" --wipe-bcache"
  }
  if WriteBack {
    bcache_cmd = bcache_cmd+" --writeback"
  }
  out, err := RunSystemCommand(bcache_cmd)
  fmt.Println(out)
  already_formatted, _ := regexp.MatchString("Already a bcache device", out)
  if already_formatted {
    fmt.Println("This format attempt probably registered the existing bcache device and it will show up using:")
    fmt.Printf("  bcachectl list\n\nIf you really want to format it, unregister it and then use the --wipe-bcache flag:\n  bcachectl unregister {device}\n  bcachectl --wipe-bcache format -(B|C) {device}\n\n") 
  }
  if err != nil {
    os.Exit(1)
  }
  return
}
