package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/qeedquan/go-media/debug/ti/coff"
)

func main() {
	log.SetPrefix("nm:")
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 1 {
		usage()
	}
	err := nm(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: [options] file")
	flag.PrintDefaults()
	os.Exit(2)
}

func nm(name string) error {
	f, err := coff.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	var ys []*coff.Symbol
	for i := 0; i < len(f.Symbols); i++ {
		y := f.Symbols[i]
		if y.Value != 0 {
			ys = append(ys, y)
		}
		i += int(y.Aux)
	}
	sort.SliceStable(ys, func(i, j int) bool {
		if ys[i].Name == ys[j].Name {
			return ys[i].Value < ys[j].Value
		}
		return ys[i].Name < ys[j].Name
	})

	for _, y := range ys {
		secttype := '?'
		fmt.Printf("%08x %c %s\n", y.Value, secttype, y.Name)
	}

	return nil
}
