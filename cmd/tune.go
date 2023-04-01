package cmd

import (
	//"errors"
	//"fmt"
	"bcachectl/pkg/bcache"
	"github.com/spf13/cobra"
	//"gopkg.in/yaml.v2"
	//"io/ioutil"
	//"os"
	//"strconv"
	//"strings"
)

var tuneCmd = &cobra.Command{
	Use:   "tune [{bcacheN} {tunable:value}] | [from-file /some/config/file]",
	Short: "Change tunable for a bcache device or tune devices from a config file",
	Long:  "Tune bcache by writing to sysfs. Using 'from-file /file/name' will read tunables from a config file and tune each specified device or 'all' devices. Allowed tunables are:\n" + bcache.ALLOWED_TUNABLES_DESCRIPTIONS,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if IsAdmin {
			all := bcache.AllDevs()
			if args[0] == "from-file" {
				all.TuneFromFile(args[1])
			} else {
				all.RunTune(args[0], args[1])
			}
		}
	},
}
