// find a binary file in a list of files
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("inbin: ")

	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 2 {
		usage()
	}

	bin, err := os.ReadFile(flag.Arg(0))
	check(err)

	for i := 1; i < flag.NArg(); i++ {
		name := flag.Arg(i)
		in, err := os.ReadFile(name)
		check(err)

		fmt.Printf("%v: %v\n", name, bytes.Index(bin, in))
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: [options] bin in ...")
	flag.PrintDefaults()
	os.Exit(2)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
