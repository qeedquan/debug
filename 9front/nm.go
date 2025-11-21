package main

import (
	"debug/plan9obj"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

var (
	symbols = "TtDdBb"
	order   = 'a'

	status = 0
)

func main() {
	flag.Bool("a", false, "print all symbols")
	flag.Bool("g", false, "print only global symbols")
	flag.Bool("n", false, "sort according to address of symbols")
	flag.Bool("s", false, "don't sort; print in symbol-table order")
	flag.Bool("u", false, "print only undefined symbols")

	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
	}
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			symbols = "UTtLlDdBbapzZf"
		case "g":
			symbols = "TLDB"
		case "n", "s":
			order = rune(f.Name[0])
		case "u":
			symbols = "U"
		}
	})

	for _, name := range flag.Args() {
		ek(nm(name))
	}
	os.Exit(status)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: nm [options] file ...")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "\nSymbols:")
	fmt.Fprintln(os.Stderr, "  U undefined symbol")
	fmt.Fprintln(os.Stderr, "  T text segment symbol")
	fmt.Fprintln(os.Stderr, "  t static text segment symbol")
	fmt.Fprintln(os.Stderr, "  L leaf function text segment symbol")
	fmt.Fprintln(os.Stderr, "  l static leaf function text segment symbol")
	fmt.Fprintln(os.Stderr, "  D data segment symbol")
	fmt.Fprintln(os.Stderr, "  d static data segment symbol")
	fmt.Fprintln(os.Stderr, "  B bss segment symbol")
	fmt.Fprintln(os.Stderr, "  b static bss segment symbol")
	fmt.Fprintln(os.Stderr, "  a automatic (local) variable symbol")
	fmt.Fprintln(os.Stderr, "  p function parameter symbol")
	fmt.Fprintln(os.Stderr, "  z source file name")
	fmt.Fprintln(os.Stderr, "  Z source file line offset")
	fmt.Fprintln(os.Stderr, "  f source file name components")
	os.Exit(2)
}

func ek(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "nm: %v\n", err)
		status = 1
	}
}

func nm(name string) error {
	r, err := os.Open(name)
	if err != nil {
		return err
	}
	defer r.Close()

	f, err := plan9obj.NewFile(r)
	if err != nil {
		return fmt.Errorf("%v: %v", name, err)
	}

	p, err := f.Symbols()
	if err != nil {
		return fmt.Errorf("%v: %v", name, err)
	}
	sort.SliceStable(p, func(i, j int) bool {
		switch order {
		case 'n':
			return p[i].Value < p[j].Value
		case 's':
			return i < j
		default:
			return p[i].Name < p[j].Name
		}
	})

	for _, s := range p {
		if strings.Contains(symbols, string(s.Type)) {
			if symbols == "TtDdBb" && (strings.HasPrefix(s.Name, "$") || strings.HasPrefix(s.Name, ".")) {
				continue
			}

			if flag.NArg() > 1 {
				fmt.Printf("%s: %8x %c %s\n", name, s.Value, s.Type, s.Name)
			} else {
				fmt.Printf("%8x %c %s\n", s.Value, s.Type, s.Name)
			}
		}
	}

	return nil
}
