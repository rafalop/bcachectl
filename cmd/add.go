package cmd
import (

  "github.com/spf13/cobra"
  "fmt"
  "regexp"
  "os"
)

var addCmd = &cobra.Command{
  Use:   "add -[B|C] {device1} -[B|C] {device2} ... -[B|C] {deviceN}",
  Short: "add (format) bcache backing and/or cache device(s)",
  Long: "Add/Format/Create one or more bcache devices, potentially auto attaching a cache device to a backing device if both are specified together (-B) and (-C). This is a wrapper for `make-bcache` and will use the same arguments, eg. -B {backing dev} -C {cache dev}",
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
    out,_ := RunSystemCommand(`/sbin/wipefs -a `+newbdev)
    fmt.Println(out)
    out,_ = RunSystemCommand(`/sbin/wipefs -a `+newcdev)
    fmt.Println(out)
    bcache_cmd = bcache_cmd+" --wipe-bcache"
  }
  if WriteBack {
    bcache_cmd = bcache_cmd+" --writeback"
  }
  out, err := RunSystemCommand(bcache_cmd)
  fmt.Println(out)
  // we also have to register the cache dev
  if err == nil {
    RunRegister([]string{newcdev})
  }
  already_formatted, _ := regexp.MatchString("Already a bcache device", out)
  busy, _ := regexp.MatchString("Device or resource busy", out)
  if busy {
    fmt.Println("Device is busy - is it already registered bcache dev or mounted?")
  }
  if already_formatted {
    fmt.Println("This format attempt probably registered the existing bcache device and it will show up using:")
    fmt.Printf("  bcachectl list\n\nIf you REALLY want to format it, unregister it and then use the --wipe-bcache flag (will erase any superblocks and filesystems!):\n  bcachectl unregister {device}\n  bcachectl add -(B|C) {device} --wipe-bcache\n\n")
  }
  if err != nil {
    os.Exit(1)
  }
  return
}
