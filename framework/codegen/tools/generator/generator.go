package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Println("error: you must specify a path to a generate.hcl file")
		os.Exit(1)
	}

	generatefile := flag.Arg(0)
	config, err := parseGeneratorHCLConfig(generatefile)
	if err != nil {
		fmt.Printf("error parsing %v: %v\n", generatefile, err)
		os.Exit(1)
	}

	fmt.Printf("%#v", config)
}
