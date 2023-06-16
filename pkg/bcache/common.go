package bcache

import (
	"errors"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// devices in /dev/bcache/by-uuid seems to be flaky
// const BDEVS_DIR = `/dev/bcache/by-uuid/`
const SYSFS_BCACHE_ROOT = `/sys/fs/bcache/`
const SYSFS_BLOCK_ROOT = `/sys/block/`
const (
	NONE_ATTACHED = "no cache"
)

var STATS = []string{
	`state`,
	`stats_total/bypassed`,
	`stats_total/cache_hits`,
	`stats_total/cache_misses`,
	`stats_total/cache_hit_ratio`,
	`stats_total/cache_bypass_hits`,
	`stats_total/cache_bypass_misses`,
	`cache/congested`,
	`writeback_percent`,
	`dirty_data`,
}

var TUNABLES = []string{
	`cache_mode`,
	`cache/cache0/cache_replacement_policy`,
	`cache/congested_write_threshold_us`,
	`cache/congested_read_threshold_us`,
	`readahead_cache_policy`,
	`sequential_cutoff`,
	`writeback_delay`,
}

var PARAMETERS = append(STATS, TUNABLES...)

// A bcache (backing) device
type Bcache_bdev struct {
	BcacheDev  string   `json:"BcacheDev"`
	ShortName  string   `json:"ShortName"`
	BackingDev string   `json:"BackingDev"`
	CacheDev   string   `json:"CacheDev"`
	BUUID      string   `json:"BcacheDevUUID"`
	CUUID      string   `json:"CacheSetUUID"`
	Slaves     []string `json:"Devices"`
	// This map will contain extended info about bcache device, eg. stats, tunables etc
	Parameters map[string]interface{}
}

// A bcache cache device
type Bcache_cdev struct {
	Dev  string `json:"device"`
	UUID string `json:"UUID"`
}

// Struct to hold all bcache formatted devices
type BcacheDevs struct {
	Bdevs []Bcache_bdev
	Cdevs []Bcache_cdev
}

func AllDevs() (all *BcacheDevs, err error) {
	all = new(BcacheDevs)
	if err = all.FindBDevs(); err != nil {
		return
	}
	if err = all.FindCDevs(); err != nil {
		return
	}
	return all, nil
}

func RunSystemCommand(cmd string) (out string, err error) {
	cmd_split := strings.Fields(cmd)
	head := cmd_split[0]
	tail := cmd_split[1:]
	c := exec.Command(head, tail...)
	out_b, err := c.CombinedOutput()
	out = string(out_b)
	return
}

// read raw value from sysfs
func readVal(path string) (val string) {
	data, err := ioutil.ReadFile(path)
	if err != nil || string(data) == "\n" {
		val = ""
		return
	} else {
		val = strings.TrimRight(string(data), "\n")
	}
	return
}

// return current value for a bcache parameter
func (b *Bcache_bdev) Val(name string) (val string) {
	path := SYSFS_BLOCK_ROOT + `/` + b.ShortName + `/bcache/`
	// todo put all tunables in single array with full path
	//for _, p := range CACHE_TUNABLES {
	//	if name == p {
	//		path = path + `cache/`
	//	}
	//}
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

// Find backing and cache devs for a bcache set
func (b *Bcache_bdev) FindBackingAndCacheDevs() {
	// Note, newer kernels/bcache verisons seem to have 'backing_dev_name' entry in sysfs
	// which can be read to obtain the backing dev, eaiser than this process for finding it
	search_path := SYSFS_BLOCK_ROOT + b.ShortName + `/slaves/`
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

func GetSuperBlock(dev string) (string, error) {
	cmd := `bcache-super-show `
	cmd = cmd + dev
	return RunSystemCommand(cmd)
}

// Get cache set uuid
func (b *Bcache_bdev) FindCUUID() {
	cset_path, _ := filepath.EvalSymlinks(SYSFS_BLOCK_ROOT + b.ShortName + `/bcache/cache`)
	cset_path_a := strings.Split(cset_path, "/")
	b.CUUID = cset_path_a[len(cset_path_a)-1]
	//If it's empty, we try to get from superblock instead
	if b.CUUID == "" {
		super, _ := GetSuperBlock(b.BackingDev)
		re := regexp.MustCompile(`cset\.uuid[\ |\t]*([a-zA-Z0-9\-]*)`)
		found := re.FindStringSubmatch(super)
		// None found
		if len(found) == 0 || found != nil || found[1] == "00000000-0000-0000-0000-000000000000" {
			b.CUUID = NONE_ATTACHED
			b.CacheDev = NONE_ATTACHED
			return
		} else {
			b.CUUID = found[1]
		}
	}
}

func (b *Bcache_bdev) FindBUUID() {
	uuid_path, _ := filepath.EvalSymlinks(SYSFS_BLOCK_ROOT + b.ShortName + `/bcache/backing_dev_uuid`)
	b.BUUID = readVal(uuid_path)
}

// Find all formatted cache devices (may not be part of bcache set)
func (b *BcacheDevs) FindCDevs() (err error) {
	entries, err := os.ReadDir(SYSFS_BCACHE_ROOT)
	if err != nil {
		return
	}
	c := make(chan Bcache_cdev)
	count := 0
	for _, j := range entries {
		if j.Type() == fs.ModeDir && j.Name() != `.` {
			count++
			go func(entry os.DirEntry) {
				//Dodgy way to discover what the system dev is
				system_dev_link, _ := os.Readlink(SYSFS_BCACHE_ROOT + entry.Name() + `/cache0`)
				system_dev_a := strings.Split(system_dev_link, "/")
				system_dev := system_dev_a[len(system_dev_a)-2]
				c <- Bcache_cdev{Dev: `/dev/` + system_dev, UUID: entry.Name()}
			}(j)
		}
	}
	for i := 0; i < count; i++ {
		b.Cdevs = append(b.Cdevs, <-c)
	}
	return
}

// Find all bcache devices with settings and metadata
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
	c := make(chan Bcache_bdev, len(devs))
	for _, j := range devs {
		go func(entry os.DirEntry, basedir string) {
			var b Bcache_bdev
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
			b.MakeParameters(PARAMETERS)
			c <- b
		}(j, basedir)
	}
	for range devs {
		b.Bdevs = append(b.Bdevs, <-c)
	}
	return
}

// Make the params map and gather the various bcache settings/stats
func (b *Bcache_bdev) MakeParameters(vals []string) {
	//If val is in subdir
	b.Parameters = make(map[string]interface{})
	for _, val := range vals {
		if strings.Contains(val, "/") {
			val_a := strings.Split(val, "/")
			b.Parameters[string(val_a[len(val_a)-1])] = b.Val(val)
		} else {
			b.Parameters[val] = b.Val(val)
		}
	}
	return
}

func (b *BcacheDevs) IsBDevice(dev string) (ret bool, ret2 Bcache_bdev) {
	ret = false
	for _, bdev := range b.Bdevs {
		if bdev.ShortName == dev ||
			bdev.BcacheDev == dev ||
			bdev.BackingDev == dev {
			ret = true
			ret2 = bdev
		}
	}
	return
}

func (b *BcacheDevs) IsCDevice(dev string) (ret bool, ret2 Bcache_cdev) {
	ret = false
	for _, cdev := range b.Cdevs {
		if cdev.Dev == dev || cdev.UUID == dev {
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
	out, err = RunSystemCommand(`wipefs -a ` + device)
	return
}

func (b *BcacheDevs) Create(newbdev string, newcdev string, wipe bool, writeback bool) (returnErr error) {
	bcache_cmd := `make-bcache`
	var out string
	if newcdev != "" {
		bcache_cmd = bcache_cmd + ` -C ` + newcdev
		if wipe {
			b.Stop(newcdev)
			_, returnErr = Wipe(newcdev)
			if returnErr != nil {
				return
			}
		}
	}
	if newbdev != "" {
		bcache_cmd = bcache_cmd + ` -B ` + newbdev
		if wipe {
			b.Stop(newbdev)
			_, returnErr = Wipe(newbdev)
			if returnErr != nil {
				return
			}
		}
	}
	if writeback {
		bcache_cmd = bcache_cmd + " --writeback"
	}
	out, err := RunSystemCommand(bcache_cmd)
	if err == nil {
		if newbdev != "" {
			returnErr = Register(newbdev)
		}
		if newcdev != "" {
			returnErr = Register(newcdev)
		}
		return
	}
	// out includes error from the executed command
	already_formatted, _ := regexp.MatchString("Already a bcache device", out)
	busy, _ := regexp.MatchString("Device or resource busy", out)
	existing_super, _ := regexp.MatchString("non-bcache superblock", out)
	if busy {
		returnErr = errors.New("Device is busy - is it already a registered bcache dev or mounted?")
	}
	if already_formatted || existing_super {
		returnErr = errors.New("An existing superblock was found on this block device, which means it is either an existing bcache device or has a filesystem on it. If you REALLY want to format this device, make sure it is not registered and use the --wipe-super flag (will erase ANY superblocks and filesystems!)\n")
	}
	return
}

// Try to register a bcache device, do nothing if already registered
func Register(device string) (returnErr error) {
	var write_path string
	write_path = SYSFS_BCACHE_ROOT + `register`
	all, returnErr := AllDevs()
	if returnErr != nil {
		return
	}
	//registered := false
	if x, _ := all.IsBDevice(device); x {
		return
	}
	if x, _ := all.IsCDevice(device); x {
		return
	}
	if CheckSysfsFor(device) {
		return
	}
	returnErr = ioutil.WriteFile(write_path, []byte(device), 0)
	if returnErr != nil {
		return
	}
	all, returnErr = AllDevs()
	if returnErr != nil {
		return
	}
	isBdev, _ := all.IsBDevice(device)
	isCdev, _ := all.IsCDevice(device)
	if !isBdev && !isCdev {
		returnErr = errors.New("Couldn't register device. Is it a formatted bcache device?")
	}
	return
}

// Stop (unregister) a bcache device
func (b *BcacheDevs) Stop(device string) (returnErr error) {
	var write_path string
	sn := strings.Split(device, "/")
	shortName := sn[len(sn)-1]
	regexpString := `[0-9]+`
	matched, err := regexp.Match(regexpString, []byte(shortName))
	if err != nil {
		returnErr = err
		return
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
		returnErr = err
		return
	}
	// wait up to 5 seconds for device to disappear, else exit without guarantees
	sysfs_path = sysfs_path + `/bcache`
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(sysfs_path); os.IsNotExist(err) {
			return
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
	return errors.New("Device was stopped but it may still be in sysfs.")
}

func (b *BcacheDevs) Unregister(device string) (returnErr error) {
	if x, _ := b.IsBDevice(device); x {
		returnErr = b.UnregisterBacking(device)
	} else if x, _ := b.IsCDevice(device); x {
		returnErr = b.UnregisterCache(device)
	} else {
		returnErr = errors.New(device + " does not appear to be a registered bcache device.")
	}
	return
}

func (b *BcacheDevs) UnregisterBacking(device string) (returnErr error) {
	var write_path string
	if x, bdev := b.IsBDevice(device); x {
		write_path = SYSFS_BLOCK_ROOT + bdev.ShortName + `/bcache/stop`
		returnErr = ioutil.WriteFile(write_path, []byte{1}, 0)
	} else {
		returnErr = errors.New(device + " does not appear to be a registered bcache BACKING device.")
	}
	return
}
func (b *BcacheDevs) UnregisterCache(device string) (returnErr error) {
	var write_path string
	if x, cdev := b.IsCDevice(device); x {
		write_path = SYSFS_BCACHE_ROOT + cdev.UUID + `/stop`
		returnErr = ioutil.WriteFile(write_path, []byte{1}, 0)
	} else {
		returnErr = errors.New(device + " does not appear to be a registered bcache CACHE device.")
	}
	return
}

// Attach cache device cdev to backing dev bdev. the bdev can be either an original system device
// or a registered 'bcacheX' device
func (b *BcacheDevs) Attach(cdev string, bdev string) (returnErr error) {
	var x bool
	var y Bcache_bdev
	var z Bcache_cdev
	if x, y = b.IsBDevice(bdev); !x {
		return errors.New(bdev + " does not appear to be a formatted and registered BACKING device.")
	}
	if x, z = b.IsCDevice(cdev); !x {
		return errors.New(cdev + " does not appear to be a formatted and registered CACHE device.")
	}
	write_path := SYSFS_BLOCK_ROOT + y.ShortName + `/bcache/attach`
	ioutil.WriteFile(write_path, []byte(z.UUID), 0)
	y.FindCUUID()
	if y.CUUID != z.UUID {
		returnErr = errors.New("Cache device could not be attached. Is there already a cache set associated with the device?\n")
		return
	}
	return
}

// Detach cdev cache device from bdev bcache device, bdev can be either original system device
// or a registered 'bcacheX' device
func (b *BcacheDevs) Detach(cdev string, bdev string) (returnErr error) {
	var writepath string = SYSFS_BLOCK_ROOT
	var x bool
	var y Bcache_cdev
	var z Bcache_bdev
	x, y = b.IsCDevice(cdev)
	if !x {
		return errors.New(cdev + " is not a registered cache device.")
	}
	x, z = b.IsBDevice(bdev)
	if !x {
		return errors.New(bdev + " is not a registered backing device.")
	}
	writepath = writepath + z.ShortName + `/bcache/detach`
	returnErr = ioutil.WriteFile(writepath, []byte(y.UUID), 0)
	return
}

// Helper to check sysfs for a bcache device (means kernel already knows about the device)
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

	// Check for sysfs path a couple of times (udev is meant to auto register)
	for i := 0; i < 2; i++ {
		if _, err := os.Stat(sysfsPath); !os.IsNotExist(err) {
			return true
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
	return false
}

// Check sysfs that bcache kernel module is loaded
func BcacheModuleLoaded() bool {
	if _, err := os.Stat(SYSFS_BCACHE_ROOT); os.IsNotExist(err) {
		return false
	}
	return true
}

// Flush a single device. The only way to do this is to set the cache mode to adjust
// the cache_mode and the writeback_delay. Return two errors, first is whether we actually
// were able to complete the flush, second is setting mode back to writeback
func (b *Bcache_bdev) FlushCache() (error, error) {
	var err, err2 error
	// First check if current mode is writeback
	r := b.Val(`cache_mode`)
	if r != "writeback" {
		// nothing to flush
		return nil, nil
	}

	// Set writeback_delay to something short
	write_delay := b.Val(`writeback_delay`)
	err = b.ChangeTunable(`writeback_delay`, `1`)
	if err != nil {
		return errors.New("error setting writeback_delay: " + err.Error()), nil
	}

	// To achieve flush, we set cachemode to writethrough until state is clean
	err = b.ChangeTunable(`cache_mode`, `writethrough`)
	if err != nil {
		return errors.New("error setting writethrough for flush: " + err.Error()), nil
	}
	tries := 0
	for {
		if tries == 30 {
			err = errors.New("could not complete flush within 30 seconds")
			break
		}
		state := b.Val(`state`)
		if state == `clean` {
			err = nil
			break
		} else {
			time.Sleep(1 * time.Second)
		}
		tries += 1
	}

	// If we called this function and got this far, cache mode must have been writeback, we change it back
	err2 = b.ChangeTunable(`cache_mode`, `writeback`)
	if err2 != nil {
		err2 = errors.New("unable to set mode back to writeback: " + err2.Error())
	}
	// Set original writeback delay
	err2 = b.ChangeTunable(`writeback_delay`, write_delay)
	if err2 != nil {
		err2 = errors.New("unable to set writeback_delay back to original value: " + err2.Error())
	}
	return err, err2
}
