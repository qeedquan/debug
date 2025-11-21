// given an offset file, generate a struct that matches the offsets
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
)

type Symbol struct {
	Name string
	Type string
	Off  int64
}

var flags struct {
	structname string
	base       int64
	lang       string
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("mksfo: ")

	parseflags()

	syms, err := getsym(flag.Arg(0))
	check(err)

	if flags.base < 0 && len(syms) > 0 {
		flags.base = syms[0].Off
	}
	if flags.base < 0 {
		log.Fatal("failed to get valid base")
	}

	switch flags.lang {
	case "go":
		gengo(flags.structname, syms, flags.base)
	default:
		log.Fatalf("unsupported language %q", flags.lang)
	}
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func parseflags() {
	flag.Int64Var(&flags.base, "b", -1, "specify base offset")
	flag.StringVar(&flags.lang, "l", "go", "specify language")
	flag.StringVar(&flags.structname, "n", "G", "specify struct name")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: [options] <offset_file>")
	flag.PrintDefaults()
	os.Exit(2)
}

func getsym(name string) ([]Symbol, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := []Symbol{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		var p Symbol

		l := s.Text()
		n, _ := fmt.Sscanf(l, "%v %v %v", &p.Name, &p.Type, &p.Off)
		switch n {
		case 2:
			p.Off = -1
			fallthrough
		case 3:
			r = append(r, p)
		}
	}

	sort.SliceStable(r, func(i, j int) bool {
		if r[i].Off < 0 || r[j].Off < 0 {
			return false
		}
		return r[i].Off < r[j].Off
	})

	return r, nil
}

func gengo(structname string, syms []Symbol, base int64) {
	unk := 1
	pos := base
	fmt.Printf("type %s struct {\n", structname)
	for i, p := range syms {
		pad := int64(0)
		if p.Off > 0 {
			pad = p.Off - pos
		}
		if pad > 0 {
			fmt.Printf("\tUnk%d [%#x]byte\n", unk, pad)
			pos += pad
			unk += 1
		}

		fmt.Printf("\t%s %s\n", p.Name, p.Type)

		size := typesize(p.Type)
		if size < 0 {
			log.Fatalf("unsupported type %q", p.Type)
		}

		pos += size
		if i+1 < len(syms) && pos > syms[i+1].Off {
			log.Fatalf("struct field %d %q overlaps: %#x <> %#x", i+1, p.Name, pos, syms[i+1].Off)
		}
	}
	fmt.Printf("}\n")
}

func typesize(typ string) int64 {
	var (
		size     int64
		basetype string
	)

	n, _ := fmt.Sscanf(typ, "[%v]%v", &size, &basetype)
	if n != 2 {
		size = 1
		basetype = typ
	}

	switch basetype {
	case "uint8", "int8", "byte":
		size *= 1
	case "uint16", "int16":
		size *= 2
	case "uint32", "int32":
		size *= 4
	case "uint64", "int64":
		size *= 8
	default:
		return -1
	}
	return size
}
