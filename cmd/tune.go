package cmd
import (

  "github.com/spf13/cobra"
  "fmt"
  "strings"
  "os"
  "io/ioutil"
)

var tuneCmd = &cobra.Command{
  Use:   "tune {device1} {tunable:value}",
  Short: "Change tunable for a bcache device. Works by writing to sysfs entries, value might be string or integer",
  //Long: "Allowed tunables:\n"+(strings.Join(ALLOWED_TUNABLES, "\n")),
  Long: "Change one of the allowed tunables:\n"+ALLOWED_TUNABLES_DESCRIPTIONS,
  Args: cobra.MinimumNArgs(2),
  Run: func(cmd *cobra.Command, args []string) {
    if IsAdmin {
      all := allDevs()
      all.RunTune(args[0], args[1])
    }
  },
}

func (b *bcache_devs) RunTune(device string, tunable string) {
  var x bool
  var y bcache_bdev
  if device == "" {
    fmt.Println("I need a device to work on, eg.\n bcachectl tune bcache0 cache_mode:writeback\n")
    return
  } else if x, y = b.IsBDevice(device); x == false {
    fmt.Println(device, "does not appear to be a valid bcache device. If you specified the backing or cache device directly, try using the 'bcacheX' device instead.\n")
    return
  }
  tunable_a := strings.Split(tunable, ":")
  if len(tunable_a[0]) == 0 || len(tunable_a[1]) == 0 {
    fmt.Println("Tunable does not appear to be specified properly, must be formatted as tunable:value, eg. cache_mode:writethrough\n")
    return
  } else {
      fmt.Println()
      fmt.Println("Changing tunable for", device, "("+y.ShortName+")", tunable, "\n")
      y.ChangeTunable(tunable_a[0], tunable_a[1])
  }
}

func (b *bcache_bdev) ChangeTunable(tunable string, val string) {
  write_path := SYSFS_BLOCK_ROOT+b.ShortName+`/bcache/`+tunable
  if _, err := os.Stat(write_path); err != nil {
    fmt.Println("Tunable does not appear to exist: ", tunable)
    fmt.Println("Tunable sysfs path attempted: ", write_path)
    return
  }
  for _,t := range ALLOWED_TUNABLES {
    if tunable == t {
      ioutil.WriteFile(write_path, []byte(val), 0)
      b.makeMap(OUTPUT_VALUES)
      b.PrintFullInfo("standard")
      return
    }
  }
  fmt.Println("Tunable is not in allowed tunable list. Allowed tunables are: ")
  fmt.Println(ALLOWED_TUNABLES)
  fmt.Println(ALLOWED_TUNABLES_DESCRIPTIONS)
  return
}
