package main

import (
	"fmt"
	"os"

	"github.com/k3a/gosma/config"
	"github.com/k3a/gosma/monitor"
)

func errChk(err error, desc string) {
	if err != nil {
		fmt.Fprint(os.Stderr, "ERROR "+desc+": "+err.Error()+"\n")
		os.Exit(9)
	}
}

func main() {
	fmt.Println(config.C.Name + " " + config.Buildstamp + "\n")

	if len(os.Args) == 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s <path-to-config>\n", os.Args[0])
		os.Exit(1)
	}
	configPath := os.Args[1]

	m := &monitor.Monitor{}

	err := m.LoadConfig(configPath)
	errChk(err, "loading config")

	select {} // sleep forever
}
