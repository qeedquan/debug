package main

import (
	"bytes"
	"debug/plan9obj"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
)

var (
	output = flag.String("o", "", "output to file")

	status = 0
)

func main() {
	flag.Usage = usage
	flag.Parse()

	switch n := flag.NArg(); {
	case n < 1:
		usage()
	case n == 1:
		if *output != "" {
			ek(strip(flag.Arg(0), *output))
			break
		}
		fallthrough
	default:
		for _, name := range flag.Args() {
			ek(strip(name, name))
		}
	}
	os.Exit(status)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: strip [options] file ...")
	flag.PrintDefaults()
	os.Exit(2)
}

func ek(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "strip:", err)
		status = 1
	}
}

func strip(in, out string) error {
	b, err := os.ReadFile(in)
	if err != nil {
		return err
	}

	r := bytes.NewReader(b)
	f, err := plan9obj.NewFile(r)
	if err != nil {
		return fmt.Errorf("%v: %v", in, err)
	}

	s := f.Section("data")
	if s == nil {
		return fmt.Errorf("%v: invalid binary", in)
	}

	n := int(s.Offset + s.Size)
	if n == len(b) {
		return fmt.Errorf("%v: already stripped", in)
	}
	if n <= 0 || n > len(b) {
		return fmt.Errorf("%v: strange size", in)
	}

	// syms spsz pcsz
	binary.BigEndian.PutUint32(b[4*4:], 0)
	binary.BigEndian.PutUint32(b[4*6:], 0)
	binary.BigEndian.PutUint32(b[4*7:], 0)

	return os.WriteFile(out, b[:n], 0755)
}
