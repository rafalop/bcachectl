package bcache

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const (
	TUNE_OK                 = 0
	TUNE_BAD_PATH           = 1
	TUNE_BAD_TUNABLE_STRING = 2
	TUNE_IO_ERROR           = 3
	TUNE_BAD_DEVICE         = 4
)

var ALLOWED_TUNABLES = []string{
	`sequential_cutoff`,
	`cache_mode`,
	`writeback_percent`,
	`writeback_delay`,
	`writeback_rate`,
}

// todo just have all tunables, and set path directly if it's a cache tunable
var CACHE_TUNABLES = []string{
	`congested_write_threshold_us`,
	`congested_read_threshold_us`,
}
var QUEUE_TUNABLES = []string {
	`read_ahead_kb`,
}

var ALLOWED_TUNABLES_DESCRIPTIONS = `
sequential_cutoff:<INT>  threshold for a sequential IO to bypass the cache, set using byte value, default 4.0M (4194304)"
readahead:<INT>  size of readahead that should be performed, set using byte value, default 0
writeback_percent:<INT> bcache tries to keep this amount of percentage of dirty data for writeback mode, a setting of 0 would flush the cache
cache_mode:<STR> cache mode to use, possible values writethrough, writeback, writearound, none`
var ALLOWED_TUNABLES_ERRORSTRING = fmt.Sprintf("\nAllowed tunables are: %s %s", ALLOWED_TUNABLES, CACHE_TUNABLES)

type DriveConfig map[string]string

// Defaults
func NewDriveConfig() map[string]DriveConfig {
	var cfg map[string]DriveConfig = map[string]DriveConfig{
		`all`: DriveConfig{
			`sequential_cutoff`: `4194304`,
			`writeback_percent`: `10`,
		},
	}
	return cfg
}

func Parse(d *map[string]DriveConfig, configFile string) (err error) {
	f, err := os.Open(configFile)
	if err != nil {
		return
	}
	defer f.Close()
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(d)
	return
}

// Convert string to bytes string, eg. "1.0k" to "1024"
func HumanToBytes(s string) (bytesVal string) {
	var l, n []rune
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			l = append(l, r)
		case r >= 'a' && r <= 'z':
			l = append(l, r)
		case r >= '0' && r <= '9':
			n = append(n, r)
		case r == '.':
			n = append(n, r)
		}
	}
	units := string(l)
	value := string(n)
	value_float, _ := strconv.ParseFloat(value, 32)
	if strings.Contains(units, "k") || strings.Contains(units, "K") {
		bytesVal = fmt.Sprintf("%d", int(value_float*1024))
	} else if strings.Contains(units, "m") || strings.Contains(units, "M") {
		bytesVal = fmt.Sprintf("%d", int(value_float*1024*1024))
	} else if strings.Contains(units, "g") || strings.Contains(units, "G") {
		bytesVal = fmt.Sprintf("%d", int(value_float*1024*1024*1024))
	} else if units == "" {
		bytesVal = fmt.Sprintf("%d", int(value_float))
	}
	return
}

// Example config file to use this func with, use cache set uuid to override 'all'
// or default config
// all:
//
//	sequential_cutoff: 16384
//
// cf85e0c3-cb0a-4c99-a003-b629adb0be0b:
//
//	sequential_cutoff: 8192
//
// 577e54bb-23d3-4ef3-b5f4-749d3124ed0f:
//
//	sequential_cutoff: 4096
//	writeback_percent: 20
func (b *BcacheDevs) TuneFromFile(configFile string) (err error) {
	cfg := NewDriveConfig()
	err = Parse(&cfg, configFile)
	if err != nil {
		return
	}
	for _, bdev := range b.Bdevs {
		if cfg[bdev.BUUID] != nil {
			for tunable, val := range cfg[bdev.BUUID] {
				err = bdev.Tune(tunable+`:`+val)
				if err != nil {
					return
				}
			}
		} else {
			for tunable, val := range cfg["all"] {
				err = bdev.Tune(tunable+`:`+val)
				if err != nil {
					return
				}
			}
		}
	}
	return
}

//func (b *BcacheDevs) Tune(device string, tunable string) error {
func (b *Bcache_bdev) Tune(tunable string) error {
	//var x bool
	//var y Bcache_bdev
	//if x, y = b.IsBDevice(device); x == false {
	//	return errors.New("invalid bcache device " + device)
	//}
	tunable_a := strings.Split(tunable, ":")
	if len(tunable_a[0]) == 0 || len(tunable_a[1]) == 0 {
		return errors.New("tunable string not properly formatted: "+ tunable)
	}

	var valToSet string
	if tunable_a[0] == "sequential_cutoff" || tunable_a[0] == "readahead" ||
		tunable_a[0] == "writeback_rate" {
		valToSet = HumanToBytes(tunable_a[1])
	} else {
		valToSet = tunable_a[1]
	}
	return b.ChangeTunable(tunable_a[0], valToSet)
}

func contains(a []string, s string) bool {
	for _, x := range a {
		if s == x {
			return true
		}
	}
	return false
}

func (b *Bcache_bdev) ChangeTunable(tunable string, val string) error {
	write_path := SYSFS_BLOCK_ROOT + b.ShortName + `/bcache/`
	if contains(ALLOWED_TUNABLES, tunable) {
		write_path = write_path + tunable
	} else if contains(CACHE_TUNABLES, tunable) {
		write_path = write_path + `/cache/` + tunable
	} else {
		return errors.New("tunable not in allowed list: " + tunable)
	}
	b.MakeParameters(PARAMETERS)
	if _, err := os.Stat(write_path); err != nil {
		return errors.New("tunable path does not exist: " + write_path)
	}
	return ioutil.WriteFile(write_path, []byte(val), 0)
}

// return map of current tunables
func (b *BcacheDevs) GetTunables() map[string]DriveConfig {
	output := make(map[string]DriveConfig)
	for _, bdev := range b.Bdevs {
		output[bdev.BUUID] = make(DriveConfig)
		for _, tunable := range ALLOWED_TUNABLES {
			output[bdev.BUUID][tunable] = bdev.Val(tunable)
		}
	}
	return output
}
