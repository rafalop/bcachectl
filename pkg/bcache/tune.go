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

var ALLOWED_TUNABLES = []string{
	`sequential_cutoff`,
	`readahead`,
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

var ALLOWED_TUNABLES_DESCRIPTIONS = `
sequential_cutoff:<INT>  threshold for a sequential IO to bypass the cache, set using byte value, default 4.0M (4194304)"
readahead:<INT>  size of readahead that should be performed, set using byte value, default 0
writeback_percent:<INT> bcache tries to keep this amount of percentage of dirty data for writeback mode, a setting of 0 would flush the cache
cache_mode:<STR> cache mode to use, possible values writethrough, writeback, writearound, none`

//Example config, use cache set uuid to override 'all' or default config
//all:
//  sequential_cutoff: 16384
//cf85e0c3-cb0a-4c99-a003-b629adb0be0b:
//  sequential_cutoff: 8192
//577e54bb-23d3-4ef3-b5f4-749d3124ed0f:
//  sequential_cutoff: 4096
//  writeback_percent: 20

type driveConfig map[string]string

// Defaults
var Config map[string]driveConfig = map[string]driveConfig{
	`all`: driveConfig{
		`sequential_cutoff`: `4194304`,
		`writeback_percent`: `10`,
	},
}

func parse(configFile string) {
	f, err := os.Open(configFile)
	if err != nil {
		fmt.Println("Error opening config file (will use defaults): ", configFile+": ", err)
	}
	defer f.Close()
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&Config)
	if err != nil {
		fmt.Println("Error loading values from config file: ", err)
	}
}

// Convert string to bytes string, eg. "1.0k" to "1024"
func humanToBytes(s string) (bytesVal string) {
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

func (b *BcacheDevs) TuneFromFile(configFile string) {
	parse(configFile)
	for _, bdev := range b.bdevs {
		if Config[bdev.BUUID] != nil {
			for tunable, val := range Config[bdev.BUUID] {
				b.RunTune(bdev.BcacheDev, tunable+`:`+val)
			}
		} else {
			for tunable, val := range Config["all"] {
				b.RunTune(bdev.BcacheDev, tunable+`:`+val)
			}
		}
	}
}

func (b *BcacheDevs) RunTune(device string, tunable string) {
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
		var valToSet string
		if tunable_a[0] == "sequential_cutoff" || tunable_a[0] == "readahead" {
			valToSet = humanToBytes(tunable_a[1])
		} else {
			valToSet = tunable_a[1]
		}
		err := y.ChangeTunable(tunable_a[0], valToSet)
		if err != nil {
			fmt.Println("Couldn't change tunable:", err)
			return
		}
		fmt.Println("Changed tunable for", device, "("+y.ShortName+")", tunable)
	}
	//y.PrintFullInfo("standard")
}

func (b *bcache_bdev) ChangeTunable(tunable string, val string) error {
	write_path := SYSFS_BLOCK_ROOT + b.ShortName + `/bcache/`
	for _, t := range ALLOWED_TUNABLES {
		if tunable == t {
			write_path = write_path + tunable
			b.makeMap(OUTPUT_VALUES)
		}
	}
	for _, t := range CACHE_TUNABLES {
		if tunable == t {
			write_path = write_path + `/cache/` + tunable
			b.makeMap(OUTPUT_VALUES)
		}
	}
	if _, err := os.Stat(write_path); err != nil {
		fmt.Println("Tunable does not appear to exist: ", tunable)
		return errors.New("Tunable path does not exist: " + write_path)
	} else {
		ioutil.WriteFile(write_path, []byte(val), 0)
		return nil
	}

	fmt.Println("Tunable is not in allowed tunable list. Allowed tunables are: ")
	fmt.Println(ALLOWED_TUNABLES)
	fmt.Println(CACHE_TUNABLES)
	fmt.Println(ALLOWED_TUNABLES_DESCRIPTIONS)
	return errors.New("Not allowed.")
}

// return map of current tunables
func (b *BcacheDevs) GetTunables() map[string]driveConfig {
	output := make(map[string]driveConfig)
	for _, bdev := range b.bdevs {
		//output[bdev.CUUID] = make(driveConfig)
		output[bdev.BUUID] = make(driveConfig)
		for _, tunable := range ALLOWED_TUNABLES {
			output[bdev.BUUID][tunable] = bdev.Val(tunable)
		}
	}
	return output
}
