package main

import (
	"fmt"
	"os"

	"github.com/davecheney/profile"
	"github.com/mitchellh/cli"
)

func main() {

	cfg := profile.Config{
		MemProfile:  true,
		ProfilePath: ".", // store profiles in current directory
	}

	// p.Stop() must be called before the program exits to
	// ensure profiling information is written to disk.
	p := profile.Start(&cfg)

	defer p.Stop()

	os.Exit(realMain())
}

func realMain() int {
	//log.SetOutput(ioutil.Discard)

	// Get the command line args. We shortcut "--version" and "-v" to
	// just show the version.
	args := os.Args[1:]
	for _, arg := range args {
		if arg == "-v" || arg == "--version" {
			newArgs := make([]string, len(args)+1)
			newArgs[0] = "version"
			copy(newArgs[1:], args)
			args = newArgs
			break
		}
	}

	cli := &cli.CLI{
		Args:     args,
		Commands: Commands,
		HelpFunc: cli.BasicHelpFunc("mqtt-bridgeify"),
	}

	exitCode, err := cli.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
		return 1
	}

	return exitCode
}
