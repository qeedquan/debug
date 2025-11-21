package main

import (
	"debug/plan9obj"
	"flag"
	"fmt"
	"os"
)

var (
	status = 0
)

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() == 0 {
		ek(size("a.out"))
	} else {
		for _, name := range flag.Args() {
			ek(size(name))
		}
	}
	os.Exit(status)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: size [options] [a.out ...]")
	flag.PrintDefaults()
	os.Exit(2)
}

func ek(err error) bool {
	if err != nil {
		fmt.Fprintln(os.Stderr, "size:", err)
		status = 1
		return true
	}
	return false
}

func size(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	p, err := plan9obj.NewFile(f)
	if err != nil {
		return fmt.Errorf("%v: %v", name, err)
	}

	textsz := uint32(0)
	if s := p.Section("text"); s != nil {
		textsz += s.Size
	}

	datasz := uint32(0)
	if s := p.Section("data"); s != nil {
		datasz += s.Size
	}

	fmt.Printf("%dt + %dd + %db = %d\t%s\n", textsz, datasz, p.Bss, textsz+datasz+p.Bss, name)

	return nil
}
