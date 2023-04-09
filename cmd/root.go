package cmd

import (
	"bcachectl/pkg/bcache"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/user"
)

func CheckAdmin(user *user.User) bool {
	if user.Uid != "0" {
		return false
	}
	return true
}

// Check that bcache is loaded and ready for use
func CheckSysFS() {
	if ! bcache.BcacheLoaded() {
		fmt.Println("Bcache is not in sysfs yet (" + bcache.SYSFS_BCACHE_ROOT + "), I can't do anything!")
		fmt.Printf("Check that the bcache kernel module is loaded:\n\nlsmod|grep bcache\nmodprobe bcache\n\n")
		os.Exit(1)
	}
}

// Flags
var U *user.User
var IsAdmin bool = false
var Format string //Output format
var Extra string  //Output extra values
var Wipe bool
var NewBDev string
var NewCDev string
var WriteBack bool
var ApplyToAll bool
var OutConfigFile string

var rootCmd = &cobra.Command{
	Use:   "bcachectl",
	Short: "A command line tool for simplified administration of bcache devices",
}

func Init() {
	U, _ = user.Current()
	IsAdmin = CheckAdmin(U)
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().BoolVarP(&Wipe, "wipe-super", "", false, "force deletion of existing filesystem superblock")
	addCmd.Flags().StringVarP(&NewBDev, "backing-device", "B", "", "Backing dev to create, if specified with -C, will auto attach the cache device")
	addCmd.Flags().StringVarP(&NewCDev, "cache-device", "C", "", "Cache dev to create, if specified with -B, will auto attach the cache device")
	addCmd.Flags().BoolVarP(&WriteBack, "writeback", "", false, "Use writeback caching (when auto attach specifying -B and -C)")
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&Format, "format", "f", "table", "Output format [table|json|short]")
	listCmd.Flags().StringVarP(&Extra, "extra-vals", "e", "", "Extra settings to print (comma delim)")
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(unregisterCmd)
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().StringVarP(&Format, "format", "f", "standard", "Output format [standard|json]")
	rootCmd.AddCommand(tuneCmd)
	rootCmd.AddCommand(printTunablesCmd)
	printTunablesCmd.Flags().StringVarP(&OutConfigFile, "outfile", "o", "", "Write out tunables file to this file")
	rootCmd.AddCommand(flushCmd)
	flushCmd.Flags().BoolVarP(&ApplyToAll, "all", "a", false, "flush all devices")
	//tuneCmd.Flags().BoolVarP(&ApplyToAll, "all", "a", false, "apply tune to all devices")
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(superCmd)
	rootCmd.AddCommand(detachCmd)
}

func Execute() {
	Init()
	if len(os.Args) > 1 && !IsAdmin && !(os.Args[1] == "help" || os.Args[len(os.Args)-1] == "-h" || os.Args[len(os.Args)-1] == "--help") {
		fmt.Println("bcachectl commands require root privileges\n")
		return
	}
	CheckSysFS()
	rootCmd.Execute()
}
