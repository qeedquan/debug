// PCI BDF byte encoding/decoding
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

var (
	dflag = flag.Bool("d", false, "decode")
)

func main() {
	flag.Usage = usage
	flag.Parse()
	switch *dflag {
	case false:
		if flag.NArg() < 3 {
			usage()
		}

		for i := 0; i+2 < flag.NArg(); i += 3 {
			bus, _ := strconv.ParseInt(flag.Arg(i), 0, 64)
			device, _ := strconv.ParseInt(flag.Arg(i+1), 0, 64)
			funct, _ := strconv.ParseInt(flag.Arg(i+2), 0, 64)
			fmt.Printf("%d:%d:%d: %x\n",
				bus, device, funct, packbdf(bus, device, funct))
		}

	default:
		if flag.NArg() < 1 {
			usage()
		}

		for i := 0; i < flag.NArg(); i++ {
			bdf, _ := strconv.ParseInt(flag.Arg(0), 0, 64)
			bus, device, funct := unpackbdf(bdf)
			fmt.Printf("%x: %d:%d:%d\n",
				bdf, bus, device, funct)
		}
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: bus device funct ...")
	fmt.Fprintln(os.Stderr, "       [-d] packed ...")
	flag.PrintDefaults()
	os.Exit(2)
}

func packbdf(bus, device, funct int64) int64 {
	return (bus<<16)&0x00ff0000 |
		(device<<11)&0x0000f800 |
		(funct<<8)&0x00000700
}

func unpackbdf(bdf int64) (bus, device, funct int64) {
	bus = (bdf >> 16) & 0xff
	device = (bdf >> 11) & 0x1f
	funct = (bdf >> 8) & 0x7
	return
}
