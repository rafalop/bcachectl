package main

import (
	"flag"
	mcobra "github.com/muesli/mango-cobra"
	"github.com/muesli/roff"
	"github.com/rafalop/bcachectl/cmd"
	"os"
)

func main() {
	var outFile = flag.String("o", "bcachectl.man.8", "filename to save as manpage")
	flag.Parse()
	cmd.Init()
	r := cmd.GetRootCmd()
	manPage, err := mcobra.NewManPage(1, r)
	if err != nil {
		panic(err)
	}

	manPage = manPage.WithSection("Copyright", "(C) 2022-2023 Rafael Lopez.\n"+
		"Released under GPL-3.0 license.")

	err = os.WriteFile(*outFile, []byte(manPage.Build(roff.NewDocument())), 0644)

}
