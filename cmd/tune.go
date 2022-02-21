package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
)

var tuneCmd = &cobra.Command{
	Use:   "tune [{bcacheN} {tunable:value}] | [from-file /some/config/file]",
	Short: "Change tunable for a bcache device or tune devices from a config file",
	Long:  "Tune bcache by writing to sysfs. Using 'from-file /file/name' will read tunables from a config file and tune each specified device or 'all' devices. Allowed tunables are:\n" + ALLOWED_TUNABLES_DESCRIPTIONS,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if IsAdmin {
			all := allDevs()
			if args[0] == "from-file" {
				all.TuneFromFile(args[1])
			} else {
				all.RunTune(args[0], args[1])
			}
		}
	},
}

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

func (b *bcache_devs) TuneFromFile(configFile string) {
	parse(configFile)
	for _, bdev := range b.bdevs {
		if Config[bdev.CUUID] != nil {
			for tunable, val := range Config[bdev.CUUID] {
				b.RunTune(bdev.BcacheDev, tunable+`:`+val)
			}
		} else {
			for tunable, val := range Config["all"] {
				b.RunTune(bdev.BcacheDev, tunable+`:`+val)
			}
		}
	}
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
		err := y.ChangeTunable(tunable_a[0], tunable_a[1])
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
  fmt.Println(write_path)
	for _, t := range ALLOWED_TUNABLES {
		if tunable == t {
      write_path = write_path + tunable
			b.makeMap(OUTPUT_VALUES)
		}
	}
  for _, t := range CACHE_TUNABLES {
		if tunable == t {
      write_path = write_path+`/cache/`+tunable
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
