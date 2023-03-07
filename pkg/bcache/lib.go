package bcache

import (
	"fmt"
	"os"
	"regexp"
	"encoding/json"
	"io/fs"
	"os/exec"
	//"os/user"
	"path/filepath"
	"strings"
	"io/ioutil"
	"time"
	"unicode"
)

// This seems to be flaky
//const BDEVS_DIR = `/dev/bcache/by-uuid/`
const SYSFS_BCACHE_ROOT = `/sys/fs/bcache/`
const SYSFS_BLOCK_ROOT = `/sys/block/`

// standard values to print
var OUTPUT_VALUES = []string{
	`cache_mode`,
	`state`,
}

// extended values to print
var EXTENDED_VALUES = []string{
	`stats_total/bypassed`,
	`stats_total/cache_hits`,
	`stats_total/cache_misses`,
	`stats_total/cache_hit_ratio`,
	`stats_total/cache_bypass_hits`,
	`stats_total/cache_bypass_misses`,
	`cache/cache0/cache_replacement_policy`,
	`cache/congested`,
	`sequential_cutoff`,
	`readahead_cache_policy`,
	`writeback_percent`,
	`dirty_data`,
}


// A bcache (backing) device
type bcache_bdev struct {
	BcacheDev  string `json:"BcacheDev"`
	ShortName  string `json:"ShortName"`
	BackingDev string `json:"BackingDev"`
	CacheDev   string `json:"CacheDev"`
	BUUID string            `json:"BcacheDevUUID"`
	CUUID  string   `json:"CacheSetUUID"`
	Slaves []string `json:"Slaves"`
	// Map will contain extended info about bcache device, eg. stats etc
	Map map[string]interface{}
}

// A bcache cache device
type bcache_cdev struct {
	Dev  string `json:"device"`
	UUID string `json:"UUID"`
}

// Type to hold all bcache formatted devices
// Backing and cache devices
type BcacheDevs struct {
	bdevs []bcache_bdev
	cdevs []bcache_cdev
}

func AllDevs() BcacheDevs {
	all := new(BcacheDevs)
	if err := all.FindBDevs(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	if err := all.FindCDevs(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	return *all
}

func RunSystemCommand(cmd string) (out string, err error) {
	cmd_split := strings.Fields(cmd)
	head := cmd_split[0]
	tail := cmd_split[1:]
	c := exec.Command(head, tail...)
	out_b, err := c.CombinedOutput()
	out = string(out_b)
	//if err != nil {
	//  fmt.Println("Command returned error.")
	//}
	//fmt.Println("runcmd:", out)
	return
}

// read raw value
func readVal(path string) (val string) {
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
	path := SYSFS_BLOCK_ROOT + `/` + b.ShortName + `/bcache/`
	// todo put all tunables in single array with full path
	for _, p := range CACHE_TUNABLES {
		if name == p {
			path = path + `cache/`
		}
	}
	path = path + name
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

func getSysDevFromID(dev_id string) (path string) {
	path, _ = filepath.EvalSymlinks(`/dev/block/` + dev_id)
	return path
}

//Find backing and cache devs for a bcache set
func (b *bcache_bdev) FindBackingAndCacheDevs() {
	search_path := SYSFS_BLOCK_ROOT + b.ShortName + `/slaves/`
	//fmt.Println(b.Slaves)
	for _, slave := range b.Slaves {
		if _, registerCheck := os.Stat(search_path + slave + `/bcache`); os.IsNotExist(registerCheck) {
			b.BackingDev = "UNREGISTERED"
			b.CacheDev = "UNREGISTERED"
			break
		}
		dents, _ := os.ReadDir(search_path + slave + `/bcache`)
		for _, entry := range dents {
			entry_s := entry.Name()
			dev_id := readVal(search_path + slave + "/dev")
			if entry_s == "dev" {
				b.BackingDev = getSysDevFromID(dev_id)
				continue
			} else if entry_s == "set" {
				b.CacheDev = getSysDevFromID(dev_id)
				continue
			}
		}
	}
}

func GetSuperBlock(dev string) string {
	cmd := `/sbin/bcache-super-show `
	cmd = cmd + dev
	out, _ := RunSystemCommand(cmd)
	return out
}

// Get cache set uuid
func (b *bcache_bdev) FindCUUID() {
	cset_path, _ := filepath.EvalSymlinks(SYSFS_BLOCK_ROOT + b.ShortName + `/bcache/cache`)
	cset_path_a := strings.Split(cset_path, "/")
	b.CUUID = cset_path_a[len(cset_path_a)-1]
	//If it's empty, we try to get from superblock instead
	if b.CUUID == "" {
		super := GetSuperBlock(b.BackingDev)
		re := regexp.MustCompile(`cset\.uuid[\ |\t]*([a-zA-Z0-9\-]*)`)
		found := re.FindStringSubmatch(super)
		// None found
		if len(found) == 0 || found != nil || found[1] == "00000000-0000-0000-0000-000000000000" {
			b.CUUID = "(none attached)"
			b.CacheDev = "(none attached)"
			return
		} else {
			b.CUUID = found[1]
		}
	}
}

func (b *bcache_bdev) FindBUUID() {
	uuid_path, _ := filepath.EvalSymlinks(SYSFS_BLOCK_ROOT + b.ShortName + `/bcache/backing_dev_uuid`)
	b.BUUID = readVal(uuid_path)
}

//Find all formatted cache devices (may not be part of bcache set)
func (b *BcacheDevs) FindCDevs() (err error) {
	entries, err := os.ReadDir(SYSFS_BCACHE_ROOT)
	if err != nil {
		return
	}
	c := make(chan bcache_cdev)
	count := 0
	for _, j := range entries {
		if j.Type() == fs.ModeDir && j.Name() != `.` {
			count++
			go func(entry os.DirEntry) {
				//Dodgy way to discover what the system dev is
				system_dev_link, _ := os.Readlink(SYSFS_BCACHE_ROOT + entry.Name() + `/cache0`)
				system_dev_a := strings.Split(system_dev_link, "/")
				system_dev := system_dev_a[len(system_dev_a)-2]
				c <- bcache_cdev{Dev: `/dev/` + system_dev, UUID: entry.Name()}
			}(j)
		}
	}
	for i := 0; i < count; i++ {
		b.cdevs = append(b.cdevs, <-c)
	}
	return
}

//Find all bcache devices with settings and metadata
func (b *BcacheDevs) FindBDevs() (err error) {
	var basedir string
	var devs []os.DirEntry
	// This seems to be flaky for some reason, udevadm? we just use /dev/bcacheX
	//if _, basedirCheck := os.Stat(BDEVS_DIR); ! os.IsNotExist(basedirCheck) {
	//  fmt.Println("Found BDEVS_DIR")
	//  devs, err = os.ReadDir(BDEVS_DIR)
	//  basedir = BDEVS_DIR
	//} else {
	basedir = `/dev/`
	dents, err2 := os.ReadDir(basedir)
	if err2 != nil {
		err = err2
		return
	}
	for _, x := range dents {
		matched, _ := regexp.Match(`bcache[0-9]+`, []byte(x.Name()))
		if matched {
			devs = append(devs, x)
		}
	}
	//}
	c := make(chan bcache_bdev, len(devs))
	for _, j := range devs {
		go func(entry os.DirEntry, basedir string) {
			var b bcache_bdev
			//todo fix this variable name, not really uuid
			uuid_path := basedir + entry.Name()
			bcache_device, err2 := filepath.EvalSymlinks(uuid_path)
			if err2 != nil {
				err = err2
				return
			}
			sn := strings.Split(bcache_device, "/")
			b.ShortName = sn[len(sn)-1]
			slave_dents, _ := os.ReadDir(SYSFS_BLOCK_ROOT + b.ShortName + `/slaves`)
			for _, j := range slave_dents {
				b.Slaves = append(b.Slaves, j.Name())
			}
			b.FindBackingAndCacheDevs()
			b.FindCUUID()
			b.BcacheDev = bcache_device
			b.FindBUUID()
			//b.CacheMode = b.Val(`cache_mode`)
			b.makeMap(OUTPUT_VALUES)
			c <- b
		}(j, basedir)
	}
	for range devs {
		b.bdevs = append(b.bdevs, <-c)
	}
	return
}

func (b *bcache_bdev) makeMap(vals []string) {
	var m = make(map[string]interface{})
	m["BcacheDev"] = b.BcacheDev
	m["BackingDev"] = b.BackingDev
	m["CacheDev"] = b.CacheDev
	for _, val := range vals {
		m[val] = b.Val(val)
	}
	b.Map = m
	return
}

func (b *bcache_bdev) extendMap(extra_vals []string) {
	//If val is in subdir
	for _, val := range extra_vals {
		if strings.Contains(val, "/") {
			val_a := strings.Split(val, "/")
			b.Map[string(val_a[len(val_a)-1])] = b.Val(val)
		} else {
			b.Map[val] = b.Val(val)
		}
	}
	return
}

func PrintColumn(val string) {
	fmt.Printf("%-18s", val)
}

func (b *bcache_bdev) PrintFullInfo(format string) {
	b.extendMap(EXTENDED_VALUES)
	if format == "json" {
		all_out := struct {
			ShortName    string
			BUUID        string
			CUUID        string
			ExtendedInfo map[string]interface{}
		}{
			ShortName: b.ShortName,
			BUUID: b.BUUID,
			CUUID:        b.CUUID,
			ExtendedInfo: b.Map,
		}
		json_out, _ := json.Marshal(all_out)
		fmt.Println(string(json_out))
	} else {
		fmt.Printf("%-30s%s\n", "ShortName:", b.ShortName)
		fmt.Printf("%-30s%s\n", "Bcache Dev UUID:" , b.BUUID)
		fmt.Printf("%-30s%s\n", "Cache Set UUID:", b.CUUID)
		for k, v := range b.Map {
			fmt.Printf("%-30s%s\n", k+`:`, v)
		}
	}
	return
}

func (b *BcacheDevs) printTable(extra_vals []string) {
	fmt.Println("bcache devices:")
	if len(b.bdevs) > 0 {
		columns := []string{"BcacheDev", "BackingDev", "CacheDev", "cache_mode", "state"}
		//var extra_vals []string
		//extra_vals = []string{"state", "dirty_data", "sequential_cutoff"}
		for _, val := range extra_vals {
			columns = append(columns, val)
		}
		for _, j := range columns {
			PrintColumn(j)
		}
		fmt.Printf("\n")
		//fmt.Printf("%-15s %-15s %-15s\n", "bcache_dev", "backing_dev", "cache_dev")
		for _, bdev := range b.bdevs {
			if len(extra_vals) > 0 {
				bdev.extendMap(EXTENDED_VALUES)
			}
			for _, j := range columns {
				//        bdev.extendMap(extra_vals)
				//fmt.Println(bdev.Map[j])
				if bdev.Map[j] != nil {
					PrintColumn(bdev.Map[j].(string))
				}
			}
			fmt.Printf("\n")
		}
	} else {
		fmt.Println("None found.")
	}

	fmt.Printf("\n")
	fmt.Println("registered cache devices:")
	if len(b.cdevs) > 0 {
		for _, cdev := range b.cdevs {
			fmt.Println(cdev.Dev, cdev.UUID)
		}
		fmt.Printf("\n")
	} else {
		fmt.Println("None found.")
	}
}

func (b *BcacheDevs) IsBDevice(dev string) (ret bool, ret2 bcache_bdev) {
	ret = false
	for _, bdev := range b.bdevs {
		if bdev.ShortName == dev ||
			bdev.BcacheDev == dev ||
			bdev.BackingDev == dev {
			ret = true
			ret2 = bdev
		}
	}
	return
}

func (b *BcacheDevs) IsCDevice(dev string) (ret bool, ret2 bcache_cdev) {
	ret = false
	for _, cdev := range b.cdevs {
		if cdev.Dev == dev {
			ret = true
			ret2 = cdev
		}
	}
	return
}

func (b *BcacheDevs) IsBCDevice(dev string) (ret bool) {
	ret = false
	if x, _ := b.IsBDevice(dev); x {
		ret = true
	} else if x, _ := b.IsCDevice(dev); x {
		ret = true
	}
	return
}

func Wipe(device string) (out string, err error) {
	out, err = RunSystemCommand(`/sbin/wipefs -a ` + device)
	return
}

func (b *BcacheDevs) RunCreate(newbdev string, newcdev string, wipe bool, writeback bool) {
	bcache_cmd := `/usr/sbin/make-bcache`
	var out string
	if newcdev != "" {
		bcache_cmd = bcache_cmd + ` -C ` + newcdev
		if wipe {
			b.RunStop(newcdev)
			//out, _ = RunSystemCommand(`/sbin/wipefs -a ` + newcdev)
			Wipe(newcdev)
			fmt.Println(out)
		}
	}
	if newbdev != "" {
		bcache_cmd = bcache_cmd + ` -B ` + newbdev
		if wipe {
			b.RunStop(newbdev)
			//out, _ := RunSystemCommand(`/sbin/wipefs -a ` + newbdev)
			Wipe(newbdev)
			fmt.Println(out)
		}
	}
	if writeback {
		bcache_cmd = bcache_cmd + " --writeback"
	}
	out, err := RunSystemCommand(bcache_cmd)
	if err == nil {
		fmt.Println("Completed formatting device(s):", newbdev, newcdev)
		if newbdev != "" {
			RunRegister([]string{newbdev})
		}
		if newcdev != "" {
			RunRegister([]string{newcdev})
		}
	}
	// out includes error from the executed command
	already_formatted, _ := regexp.MatchString("Already a bcache device", out)
	busy, _ := regexp.MatchString("Device or resource busy", out)
	existing_super, _ := regexp.MatchString("non-bcache superblock", out)
	if busy {
		fmt.Println("Device is busy - is it already registered bcache dev or mounted?")
	}
	if already_formatted || existing_super {
		//fmt.Println(out)
		fmt.Printf("An existing superblock was found on this block device, which means it is either an existing bcache device or has a filesystem on it. If you REALLY want to format this device, make sure it is not registered and use the --wipe-super flag (will erase ANY superblocks and filesystems!)\n")
	}
	if err != nil {
		os.Exit(1)
	}
	return
}

func (b *BcacheDevs) RunStop(device string) error {
	var write_path string
	sn := strings.Split(device, "/")
	shortName := sn[len(sn)-1]
	regexpString := `[0-9]+`
	matched, err := regexp.Match(regexpString, []byte(shortName))
	if err != nil {
		fmt.Println(err)
		return err
	}
	if matched {
		topDev := strings.TrimRightFunc(shortName, func(r rune) bool {
			return unicode.IsNumber(r)
		})
		write_path = SYSFS_BLOCK_ROOT + topDev + `/` + shortName
	} else {
		write_path = SYSFS_BLOCK_ROOT + shortName
	}
	sysfs_path := write_path
	write_path = write_path + `/bcache/`
	if x, _ := b.IsCDevice(device); x {
		write_path = write_path + `set/`
	}
	write_path += `stop`

	err = ioutil.WriteFile(write_path, []byte{1}, 0)
	if err != nil {
		fmt.Println(err)
		return err
	}
	// wait up to 5 seconds for device to disappear, else exit without guarantees
	sysfs_path = sysfs_path + `/bcache`
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(sysfs_path); os.IsNotExist(err) {
			fmt.Println(device, "bcache sysfs path now removed by kernel.")
			return nil
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
	fmt.Println(device, "is still formatted and may still be in sysfs")
	return nil
}

func RunRegister(devices []string) {
	var write_path string
	write_path = SYSFS_BCACHE_ROOT + `register`
	all := AllDevs()
	for _, device := range devices {
		//fmt.Println("write_path:", write_path, "device:", device)
		if x := all.IsBCDevice(device); x {
			fmt.Println(device, "is already registered.")
		} else {
			err := ioutil.WriteFile(write_path, []byte(device), 0)
			if err != nil {
				if CheckSysfsFor(device) {
					fmt.Println(device, "is already registered.")
					return
				}
				fmt.Println(err)
			}
			all = AllDevs()
			if x, y := all.IsBDevice(device); x {
				fmt.Println(device, "was registered as", y.ShortName+".")
			} else if x, y := all.IsCDevice(device); x {
				fmt.Println(device, "was registered as a cache device with uuid", y.UUID+".")
			} else {
				fmt.Println("Couldn't register device. If the device has an associated cache device, try registering the cache device instead.")
				os.Exit(1)
			}
		}
	}
	fmt.Println()
	return
}

func (b *BcacheDevs) RunList(format string, extra string) {
	extra_vals := strings.Split(extra, `,`)
	for _, j := range b.bdevs {
		j.extendMap(extra_vals)
	}
	if format == "json" {
		out := `{`
		jsonb_out, _ := json.Marshal(b.bdevs)
		jsonc_out, _ := json.Marshal(b.cdevs)
		out = out + `"BcacheDevs":` + string(jsonb_out) + `, "CacheDevs":` + string(jsonc_out) + `}`
		fmt.Println(out)
	} else if format == "short" {
		for _, bdev := range b.bdevs {
			fmt.Println(bdev.ShortName)
		}
	} else {
		b.printTable(extra_vals)
	}
	return
}

func (b *BcacheDevs) RunUnregister(devices []string) {
	var write_path string
	for _, device := range devices {
		if x, bdev := b.IsBDevice(device); x {
			//TODO only stop if it is alreayd registered
			write_path = SYSFS_BLOCK_ROOT + bdev.ShortName + `/bcache/stop`
			ioutil.WriteFile(write_path, []byte{1}, 0)
			fmt.Println(device, "(backing device) was unregistered, but is still formatted.")
			return
		}
		if x, cdev := b.IsCDevice(device); x {
			//Also here, only if registered
			write_path = SYSFS_BCACHE_ROOT + cdev.UUID + `/stop`
			ioutil.WriteFile(write_path, []byte{1}, 0)
			fmt.Println(device, "(cache device) was unregistered, but is still formatted.")
			return
		}
		fmt.Println(device + " does not appear to be a registered bcache device.")
	}
	return
}

func (b *BcacheDevs) RunShow(format string, device string) (err error) {
	if device == "" {
		fmt.Println("I need a device to show! specify one eg.\n bcachectl show bcache0\n bcachectl show /dev/sda")
		os.Exit(1)
		return
	}
	found := false
	if x, y := b.IsBDevice(device); x {
		y.PrintFullInfo(format)
		found = true
	}
	if found == false {
		fmt.Println("Device '" + device + "' is not a registered bcache device")
		os.Exit(1)
	}
	return
}

func (b *BcacheDevs) RunAttach(cdev string, bdev string) {
	var x bool
	var y bcache_bdev
	var z bcache_cdev
	if x, y = b.IsBDevice(bdev); !x {
		fmt.Println(bdev, "does not appear to be a formatted and registered BACKING device.")
		return
	}
	if x, z = b.IsCDevice(cdev); !x {
		fmt.Println(cdev, "does not appear to be a formatted and registered CACHE device.")
		return
	}
	write_path := SYSFS_BLOCK_ROOT + y.ShortName + `/bcache/attach`
	ioutil.WriteFile(write_path, []byte(z.UUID), 0)
	y.FindCUUID()
	if y.CUUID != z.UUID {
		fmt.Println("Cache device could not be attached. Is there already a cache set associated with the device?\n")
		return
	}
	fmt.Println("Cache device", cdev, "was attached as cache for", bdev, "("+y.ShortName+")")
}

func (b *BcacheDevs) RunDetach(cdev string, bdev string) {
	var writepath string = SYSFS_BLOCK_ROOT
	var x bool
	var y bcache_cdev
	var z bcache_bdev
	x, y = b.IsCDevice(cdev)
	if !x {
		fmt.Println(cdev, "is not a registered cache device.")
		return
	}
	x, z = b.IsBDevice(bdev)
	if !x {
		fmt.Println(bdev, "is not a registered backing device.")
		return
	}
	writepath = writepath + z.ShortName + `/bcache/detach`
	ioutil.WriteFile(writepath, []byte(y.UUID), 0)
	fmt.Println("Detached cache dev", cdev, "from "+bdev)
}

// Helper to check for bcache in sysfs for a device (means kernel already knows about the device)
func CheckSysfsFor(device string) bool {
	var sysfsPath string
	sn := strings.Split(device, "/")
	shortName := sn[len(sn)-1]
	regexpString := `[0-9]+`
	matched, _ := regexp.Match(regexpString, []byte(shortName))
	if matched {
		baseDev := strings.TrimRightFunc(shortName, func(r rune) bool {
			return unicode.IsNumber(r)
		})
		sysfsPath = SYSFS_BLOCK_ROOT + baseDev + `/` + shortName + `/bcache`
	} else {
		sysfsPath = SYSFS_BLOCK_ROOT + shortName + `/bcache`
	}
	fmt.Println("searching for path:" + sysfsPath)

	// Check for sysfs path a couple of times (udev is meant to auto register)
	for i := 0; i < 1; i++ {
		if _, err := os.Stat(sysfsPath); !os.IsNotExist(err) {
			fmt.Println("Found path: " + sysfsPath)
			return true
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
	return false
}

func CheckSysFS() {
	if _, err := os.Stat(SYSFS_BCACHE_ROOT); os.IsNotExist(err) {
		fmt.Println("Bcache is not in sysfs yet (" + SYSFS_BCACHE_ROOT + "), I can't do anything!")
		fmt.Printf("Check that the bcache kernel module is loaded:\n\nlsmod|grep bcache\nmodprobe bcache\n\n")
		os.Exit(1)
	}
}
