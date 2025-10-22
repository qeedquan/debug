package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 2 {
		usage()
	}

	val1, _ := strconv.ParseInt(flag.Arg(0), 0, 64)
	val2, _ := strconv.ParseInt(flag.Arg(1), 0, 64)
	id1 := int(val1 & 0xffff)
	id2 := int(val2 & 0xffff)
	fmt.Printf("OUI:   %#x\n", miioui(id1, id2))
	fmt.Printf("MODEL: %#x\n", miimodel(id2))
	fmt.Printf("REV:   %#x\n", miirev(id2))
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: id1 id2")
	flag.PrintDefaults()
	os.Exit(2)
}

func miioui(id1, id2 int) int {
	return id1<<6 | id2>>10
}

func miimodel(id2 int) int {
	return (id2 & 0x3f0) >> 4
}

func miirev(id2 int) int {
	return id2 & 0xf
}
