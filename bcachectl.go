package main

import (
	_ "syscall"
	"io/ioutil"
	"strings"
	"fmt"
	"flag"
	"os"
	"path/filepath"
	"encoding/json"
  "os/exec"
  "io/fs"
  "regexp"
)

var BDEVS_DIR = `/dev/bcache/by-uuid/`
var SYSFS_BCACHE_ROOT = `/sys/fs/bcache/`
var SYSFS_BLOCK_ROOT = `/sys/block/`
var OUTPUT_VALUES = []string{
	`cache_mode`,
	`state`,
}
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
	`readahead_cache_policy`,
	`cache_mode`,
	`cache_replacement_policy`,
}

// A bcache (backing) device
type bcache_bdev struct {
	BcacheDev string				`json:"BcacheDev"`
	ShortName string				`json:"ShortName"`
	BackingDevs []string		`json:"BackingDevs"`
	CacheDevs []string			`json:"CacheDevs"`
	BUUID string						`json:"BcacheDevUUID"`
	CUUID string						`json:"CacheSetUUID"`
	Slaves []string					`json:"Slaves"`
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

func (b *bcache_bdev) Val(name string) (val string) {
	path := SYSFS_BLOCK_ROOT+`/`+b.ShortName+`/bcache/`+name
	rawval_s := readVal(path)
	rawval_a := strings.Split(rawval_s, " ")
	if len(rawval_a) > 1 {
		for _, j := range rawval_a {
			if strings.Contains(j, `[`) &&
					strings.Contains(j, `]`) {
				val = strings.TrimLeft(j, `[`)
				val = strings.TrimRight(val, `]`)
			}
		}
	} else {
		val = rawval_a[0]
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


// Get cache set uuid
func (b *bcache_bdev) FindCUUID() {
		cset_path, _ := filepath.EvalSymlinks(SYSFS_BLOCK_ROOT+b.ShortName+`/bcache/cache`)
		cset_path_a := strings.Split(cset_path, "/")
		b.CUUID = cset_path_a[len(cset_path_a)-1]
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
        //Dodgy way to discover what dev is
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
	return
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

func (b *bcache_devs) RunList(format string) {
  if format == "json" {
    out := `{`
		jsonb_out, _ := json.Marshal(b.bdevs)
		jsonc_out, _ := json.Marshal(b.cdevs)
    out = out+`"bcache_devs":`+string(jsonb_out)+`, "cache_devs":`+string(jsonc_out)+`}`
		fmt.Println(out)
	} else if format == "short" {
		for _, bdev := range b.bdevs {
			fmt.Printf("%s ", bdev.ShortName)
		}
		fmt.Printf("\n")
	} else {
			b.printTable()
	}
  return
}

func (b *bcache_devs) RunShow(format string, device string) {
  if device == "" {
    fmt.Println("I need a device to show! specify one eg.\n bcachectl show bcache0\n bcachectl show /dev/sda")
    return
  }
	found := false
	if x, y := b.IsBDevice(device); x {
		y.PrintFullInfo(format)
    found = true
	}
	if found == false {
		fmt.Println("Device '"+device+"' is not a registered bcache device")
    return
	}
  return
}

func RunFormat(wipe bool, make_bcache_cmds []string){
  if len(make_bcache_cmds) == 0 {
    fmt.Println("I need at least one backing dev (-B) or one cache dev (-C) to format!")
    os.Exit(1)
  }
  bcache_cmd := `/usr/sbin/make-bcache `+strings.Join(make_bcache_cmds, " ")
  if wipe {
    bcache_cmd = bcache_cmd+" --wipe-bcache"
  }
  out, err := RunSystemCommand(bcache_cmd)
  fmt.Println(out)
  //out_a := strings.Split(out, " ")
  already_formatted, _ := regexp.MatchString("Already a bcache device", out)
  if already_formatted {
    fmt.Println("This format attempt probably registered the existing bcache device and it will show up using:")
    fmt.Printf("  bcachectl list\n\nIf you really want to format it, unregister it and then use the --wipe-bcache flag:\n  bcachectl unregister {device}\n  bcachectl --wipe-bcache format -(B|C){device}\n\n") 
  }
  if err != nil {
    os.Exit(1)
  }
  os.Exit(0)
}

var AVAILABLE_COMMANDS = []string{"list", "show", "register", "add", "cache"}
func printHelp(){
	//log.Println("args:", flag.Args())
  fmt.Println("Available commands:")
  fmt.Println(AVAILABLE_COMMANDS)
}
func (b *bcache_devs) RunTune(device string, tunable string) {
	tunable_a := strings.Split(tunable, ":")
	if len(tunable_a[0]) == 0 || len(tunable_a[1]) == 0 {
		fmt.Println("Tunable does not appear to be specified properly, must be formatted as tunable:value, eg. -t cache_mode:writethrough")
		os.Exit(1)
	} else {
		for _, bdev := range b.bdevs {
			if bdev.ShortName == device {
				bdev.ChangeTunable(tunable_a[0], tunable_a[1])
			}
		}
	}
}

func (b *bcache_devs) RunUnregister(devices []string){
  var write_path string
  for _, device := range devices {
    if x,bdev := b.IsBDevice(device); x {
      write_path = SYSFS_BLOCK_ROOT+bdev.ShortName+`/bcache/stop`
	    ioutil.WriteFile(write_path, []byte{1}, 0)
      fmt.Println(device, "was unregistered, but is still formatted.")
      return
    }
    if x,cdev := b.IsCDevice(device); x {
      write_path = SYSFS_BCACHE_ROOT+cdev.UUID+`/stop`
	  	ioutil.WriteFile(write_path, []byte{1}, 0)
      fmt.Println(device, "was unregistered but is still formatted.")
      return
    }
    fmt.Println(device+" does not appear to be a registered bcache device.")
  }
  return
}

func RunRegister(device string){
  var write_path string
  write_path = SYSFS_BCACHE_ROOT+`register`
  all := allDevs()
  if x := all.IsBCDevice(device); x {
    fmt.Println("Device is already registered.")
    fmt.Println()
    all.printTable()
    return
  }
	ioutil.WriteFile(write_path, []byte(device), 0)
  all = allDevs()
  done := false
  if x, _ := all.IsCDevice(device); x {
    done = true
  }
  if done {
    fmt.Println(device, "was registered.")
    all.printTable()
  } else {
    fmt.Println("Couldn't register device. If it is a backing device with cache device attached, you should try to register the cache device instead.")
  }
  return
}

//func RunAttach(bdevs, cdevs)

func main () {
  format := flag.String("f", "table", "output format for list or show commands")
  wipe_bcache := flag.Bool("wipe-bcache", false, "force format overwrite of existing bcache device")

	flag.Parse()
  all := allDevs()

  command := flag.Arg(0)

  switch command {
  case "list":
    all.RunList(*format)
  case "show":
    all.RunShow(*format, flag.Arg(1))
  case "format":
    RunFormat(*wipe_bcache, flag.Args()[1:])
//  case "attach":
//    RunAttach(flag.Arg[1], flag.Arg[2])
  case "tune":
    all.RunTune(flag.Arg(1), flag.Arg(2)) 
  case "unregister":
    all.RunUnregister(flag.Args()[1:])
  case "register":
    RunRegister(flag.Arg(1))
  default:
    fmt.Println(command, "is not an available command.")
    printHelp()
  }

}
