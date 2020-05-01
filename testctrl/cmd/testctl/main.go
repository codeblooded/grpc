package main

import (
	"fmt"
	"os"
)

const (
	// Success (exit code 0) shows the command finished without an error.
	Success = 0

	// FlagError (exit code 2) shows the command was unable to run or
	// complete due to the combination or lack of flags.
	FlagError = 2

	// ConnectionError (exit code 3) shows the command could not establish a
	// connection to services over the internet.
	ConnectionError = 3

	// SchedulingError (exit code 4) shows that the test session could not
	// be scheduled to run on the cluster.
	SchedulingError = 4
)

func exit(code int, messageFmt string, args ...interface{}) {
	fmt.Printf(messageFmt+"\n", args...)
	os.Exit(code)
}

func main() {
	subcmd := os.Args[1]
	subcmdArgs := os.Args[1:]

	switch subcmd {
	case "schedule":
		Schedule(subcmdArgs)
	default:
		fmt.Printf("Better docs are coming soon. For now, run schedule -h\n")
	}
}
