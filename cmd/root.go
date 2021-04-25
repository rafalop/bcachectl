package cmd

import (
  "github.com/spf13/cobra"
  "fmt"
  "os"
  "io/ioutil"
  "strings"
//  "flag"
  "os/user"
  "path/filepath"
  "encoding/json"
  "os/exec"
  "io/fs"
  "regexp"
  //"github.com/spf13/cobra"
)

const BDEVS_DIR = `/dev/bcache/by-uuid/`
const SYSFS_BCACHE_ROOT = `/sys/fs/bcache/`
const SYSFS_BLOCK_ROOT = `/sys/block/`

// standard values to print
var OUTPUT_VALUES = []string{
  `cache_mode`,
  `state`,
}

// extended values to print
var EXTENDED_VALUES = []string{
  `stats_total/cache_hits`,
  `stats_total/cache_misses`,
  `stats_total/cache_hit_ratio`,
  `stats_total/cache_bypass_hits`,
  `stats_total/cache_bypass_misses`,
  `cache/cache0/cache_replacement_policy`,
  `sequential_cutoff`,
  `readahead_cache_policy`,
  `writeback_percent`,
}
var ALLOWED_TUNABLES = []string{
  `sequential_cutoff`,
  `readahead`,
  `cache_mode`,
  `writeback_percent`,
}
var ALLOWED_TUNABLES_DESCRIPTIONS =`
sequential_cutoff:  threshold for a sequential IO to bypass the cache, set using byte value, default 4.0M (4194304)"
readahead:  size of readahead that should be performed, set using byte value, default 0
writeback_percent:  bcache tries to keep this amount of percentage of dirty data for writeback mode, a setting of 0 would flush the cache
cache_mode:  cache mode to use, possible values writethrough, writeback, writearound, none`

// A bcache (backing) device
type bcache_bdev struct {
  BcacheDev string        `json:"BcacheDev"`
  ShortName string        `json:"ShortName"`
  BackingDevs []string    `json:"BackingDevs"`
  CacheDevs []string      `json:"CacheDevs"`
  BUUID string            `json:"BcacheDevUUID"`
  CUUID string            `json:"CacheSetUUID"`
  Slaves []string         `json:"Slaves"`
  // Map will contain extended info about bcache device, eg. stats etc
  Map map[string]interface{}
}


// A bcache cache device
type bcache_cdev struct {
  Dev string  `json:"device"`
  UUID string `json:"UUID"`
}

// Type to hold all bcache formatted devices
// Backing and cache devices
type bcache_devs struct {
  bdevs []bcache_bdev
  cdevs []bcache_cdev
}

func allDevs() bcache_devs {
  all := new(bcache_devs)
  all.FindBDevs()
  all.FindCDevs()
  return *all
}

func RunSystemCommand(cmd string) (out string, err error){
    cmd_split := strings.Fields(cmd)
    head := cmd_split[0]
    tail := cmd_split[1:]
    c := exec.Command(head, tail ...)
    out_b, err := c.CombinedOutput()
    out = string(out_b)
    //if err != nil {
    //  fmt.Println("Command returned error.")
    //}
    //fmt.Println("runcmd:", out)
    return
}

// read raw value
func readVal(path string) (val string){
  //fmt.Println("reading val from ", path)
  data, err := ioutil.ReadFile(path)
  if err != nil || string(data) == "\n" {
    val = ""
    return
  } else {
    val = strings.TrimRight(string(data), "\n")
  }
  return
}

// return current setting for a bcache tunable
func (b *bcache_bdev) Val(name string) (val string) {
path := SYSFS_BLOCK_ROOT+`/`+b.ShortName+`/bcache/`+name
  rawval_s := readVal(path)
  if strings.Contains(rawval_s, `[`) {
    rawval_a := strings.Split(rawval_s, " ")
    if len(rawval_a) > 1 {
      for _, j := range rawval_a {
        if strings.Contains(j, `[`) &&
            strings.Contains(j, `]`) {
          val = strings.TrimLeft(j, `[`)
          val = strings.TrimRight(val, `]`)
        }
      }
    }
  } else {
    val = rawval_s
  }
  return
}

func getSysDevFromID(dev_id string) (path string){
  path, _ = filepath.EvalSymlinks(`/dev/block/`+dev_id)
  return path
}

//Find backing and cache devs for a bcache set
func (b *bcache_bdev) FindBackingAndCacheDevs() {
  search_path := SYSFS_BLOCK_ROOT+b.ShortName+`/slaves/`
  for _, slave := range b.Slaves {
    dents, _ := os.ReadDir(search_path+slave+`/bcache`)
    for _, entry := range dents {
      entry_s := entry.Name()
      dev_id := readVal(search_path+slave+"/dev")
      if entry_s == "dev" {
        b.BackingDevs = append(b.BackingDevs, getSysDevFromID(dev_id))
        continue
      } else if entry_s == "set" {
        b.CacheDevs = append(b.CacheDevs, getSysDevFromID(dev_id))
        continue
      }
    }
  }
}

func GetSuperBlock(dev string) string {
  cmd := `/usr/sbin/bcache-super-show `
  cmd = cmd+dev
  out, _ := RunSystemCommand(cmd)
  return out
}

// Get cache set uuid
func (b *bcache_bdev) FindCUUID() {
    cset_path, _ := filepath.EvalSymlinks(SYSFS_BLOCK_ROOT+b.ShortName+`/bcache/cache`)
    cset_path_a := strings.Split(cset_path, "/")
    b.CUUID = cset_path_a[len(cset_path_a)-1]
    //If it's empty, we try to get from superblock instead
    if b.CUUID == "" {
      b.CUUID = "(none attached)"
      b.CacheDevs = append(b.CacheDevs, "(none attached)")
      super := GetSuperBlock(b.BackingDevs[0])
      re := regexp.MustCompile(`cset\.uuid[\ |\t]*([a-zA-Z0-9\-]*)`)
      found := re.FindStringSubmatch(super)
      //if found != nil {
      //  fmt.Println("FOUND:", found)
      //}
      b.CUUID = found[1]+" (detached)"
    }
}

//Find all formatted cache devices (may not be part of bcache set)
func (b *bcache_devs) FindCDevs() (err error) {
  entries, err := os.ReadDir(SYSFS_BCACHE_ROOT)
  if err != nil {
    return
  }
  c := make(chan bcache_cdev)
  count := 0
  for _, j := range entries {
    if j.Type() == fs.ModeDir && j.Name() != `.`{
      count++
      go func(entry os.DirEntry) {
        //Dodgy way to discover what the system dev is
        system_dev_link,_ := os.Readlink(SYSFS_BCACHE_ROOT+entry.Name()+`/cache0`)
        system_dev_a := strings.Split(system_dev_link, "/")
        system_dev := system_dev_a[len(system_dev_a)-2]
        c <- bcache_cdev{Dev: `/dev/`+system_dev, UUID: entry.Name(),}
      }(j)
    }
  }
  for i:=0 ; i<count ; i++ {
    b.cdevs = append(b.cdevs, <-c)
  }
  return
}

//Find all bcache devices with settings and metadata
func (b *bcache_devs) FindBDevs() (err error){  
  devs, err := os.ReadDir(BDEVS_DIR)
  if err != nil {
    return
  }
  c := make(chan bcache_bdev, len(devs))
  for _, j := range devs {
    go func(entry os.DirEntry) {
      var b bcache_bdev
      uuid_path := BDEVS_DIR+entry.Name()
      bcache_device, err2 := filepath.EvalSymlinks(uuid_path)
      if err2 != nil {
        err = err2
        return
      }
      sn := strings.Split(bcache_device, "/")
      b.ShortName = sn[len(sn)-1]
      slave_dents, _  := os.ReadDir(SYSFS_BLOCK_ROOT+b.ShortName+`/slaves`)
      for _, j := range slave_dents {
        b.Slaves = append(b.Slaves, j.Name())
      }
      b.FindBackingAndCacheDevs()
      b.FindCUUID()
      b.BcacheDev = bcache_device
      b.BUUID = entry.Name()
      //b.CacheMode = b.Val(`cache_mode`)
      b.makeMap(OUTPUT_VALUES)
      c<-b
    }(j)
  }
  for range devs {
    b.bdevs = append(b.bdevs, <-c)
  }
  return
}


func (b *bcache_bdev)makeMap(vals []string) {
  var m = make(map[string]interface{})
  m["BcacheDev"] = b.BcacheDev
  m["BackingDev"] = strings.Join(b.BackingDevs, " ")
  m["CacheDev"] = strings.Join(b.CacheDevs, " ")
  for _, val := range vals {
    m[val] = b.Val(val)
  }
  b.Map = m
  return
}

func (b *bcache_bdev) extendMap(extra_vals []string) {
  //If val is in subdir
  for _, val := range extra_vals {
    if strings.Contains(val, "/"){
      val_a := strings.Split(val, "/")
      b.Map[string(val_a[len(val_a)-1])] = b.Val(val)
    } else {
      b.Map[val] = b.Val(val)
    }
  }
  return
}

func PrintColumn(val string){
  fmt.Printf("%-18s", val)
}

func (b *bcache_bdev) PrintFullInfo(format string) {
  b.extendMap(EXTENDED_VALUES)
  if format == "json" {
    all_out := struct {
      ShortName string
      BUUID string
      CUUID string
      ExtendedInfo map[string]interface{}
    }{
      ShortName: b.ShortName,
      BUUID: b.BUUID,
      CUUID: b.CUUID,
      ExtendedInfo: b.Map,
    }
    json_out, _ := json.Marshal(all_out)
    fmt.Println(string(json_out))
  } else {
    fmt.Printf("%-30s%s\n", "ShortName:" , b.ShortName)
    fmt.Printf("%-30s%s\n", "BCache Dev UUID:" , b.BUUID)
    fmt.Printf("%-30s%s\n", "Cache Set UUID:" , b.CUUID)
    for k, v := range b.Map {
      fmt.Printf("%-30s%s\n", k+`:`, v)
    }
  }
  return
}

func (b *bcache_devs) printTable() {
  fmt.Println("Registered bache (backing) devices:")
  if len(b.bdevs) > 0 {
    columns := []string{"BcacheDev", "BackingDev", "CacheDev", "cache_mode"}
    var extra_vals []string
    extra_vals = []string{"state", "dirty_data"}
    for _, val := range extra_vals {
      columns = append(columns, val)
    }
    for _,j := range columns {
      PrintColumn(j)
    }
    fmt.Printf("\n")
    //fmt.Printf("%-15s %-15s %-15s\n", "bcache_dev", "backing_dev", "cache_dev")
    for _,bdev := range b.bdevs {
      for _,j := range columns {
        bdev.extendMap(extra_vals)
        //fmt.Println(bdev.Map[j])
        PrintColumn(bdev.Map[j].(string))
      }
      fmt.Printf("\n")
    }
  } else {
    fmt.Println("None found.")
  }

  fmt.Printf("\n")
  fmt.Println("Registered cache devices:")
  if len(b.cdevs) > 0 {
    for _, cdev := range b.cdevs {
      fmt.Println(cdev.Dev, cdev.UUID)
    }
    fmt.Printf("\n")
  } else {
    fmt.Println("None found.")
  }
}


func (b *bcache_devs) IsBDevice(dev string) (ret bool, ret2 bcache_bdev){
  ret = false
  for _, bdev := range b.bdevs {
    if bdev.ShortName == dev ||
      bdev.BcacheDev == dev ||
      bdev.BackingDevs[0] == dev {
      ret = true
      ret2 = bdev
    }
  }
  return
}

func (b *bcache_devs) IsCDevice(dev string) (ret bool, ret2 bcache_cdev){
  ret = false
  for _, cdev := range b.cdevs {
    if cdev.Dev == dev {
      ret = true
      ret2 = cdev
    }
  }
  return
}

func (b *bcache_devs) IsBCDevice(dev string) (ret bool) {
  ret = false
  if x, _ := b.IsBDevice(dev); x {
    ret = true
  } else if x, _ := b.IsCDevice(dev); x {
    ret = true
  }
  return
}

func CheckAdmin(user *user.User) bool{
  if user.Uid != "0" {
    return false
  }
  return true
}


// Flags
var U *user.User
var IsAdmin bool = false
var Format string //Output format
var Wipe bool
var NewBDev string
var NewCDev string
var WriteBack bool

var rootCmd = &cobra.Command{
  Use:   "bcachectl",
  Short: "A command line tool for simplified administration of bcache devices",
}

func Init() {
  U, _ = user.Current()
  IsAdmin = CheckAdmin(U)
  rootCmd.AddCommand(listCmd)
  listCmd.Flags().StringVarP(&Format, "format", "f", "table", "Output format [table|json|short]")
  rootCmd.AddCommand(registerCmd)
  rootCmd.AddCommand(unregisterCmd)
  rootCmd.AddCommand(showCmd)
  showCmd.Flags().StringVarP(&Format, "format", "f", "standard", "Output format [standard|json]")
  rootCmd.AddCommand(createCmd)
  createCmd.Flags().BoolVarP(&Wipe, "wipe-bcache", "", false, "force reformat if device is already bcache formatted")
  createCmd.Flags().StringVarP(&NewBDev, "backing-device", "B", "", "Backing dev to create, if specified with -C, will auto attach the cache device")
  createCmd.Flags().StringVarP(&NewCDev, "cache-device", "C", "", "Cache dev to create, if specified with -B, will auto attach the cache device")
  createCmd.Flags().BoolVarP(&WriteBack, "writeback", "", false, "Cache dev to create, if specified with -B, will auto attach the cache device")
  rootCmd.AddCommand(tuneCmd)
  rootCmd.AddCommand(attachCmd)
  rootCmd.AddCommand(superCmd)
  rootCmd.AddCommand(detachCmd)
}

func Execute() {
  Init()
  if len(os.Args) > 1 && ! IsAdmin && ! (os.Args[1] == "help" || os.Args[len(os.Args)-1] == "-h" || os.Args[len(os.Args)-1] == "--help") {
    fmt.Println("bcachectl commands require root privileges\n")
    return
  }
  rootCmd.Execute()
}
