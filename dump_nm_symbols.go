package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
)

var (
	bin = flag.Bool("b", false, "binary mode")
	end = flag.String("e", "little", "endianess for binary mode")
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("dump-symbols: ")

	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
	}

	var (
		r    io.Reader
		syms []sym
		err  error
	)
	switch flag.NArg() {
	case 2:
		fd, err := os.Open(flag.Arg(0))
		ck(err)
		defer fd.Close()

		r = fd
		syms, err = getsym(flag.Arg(1))
	case 1:
		r = os.Stdin
		syms, err = getsym(flag.Arg(0))
	default:
		usage()
	}
	ck(err)

	dump(r, syms)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: dump_nm_symbols [options] <bin file> <nm file>")
	flag.PrintDefaults()
	os.Exit(2)
}

func ck(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type sym struct {
	typ   int
	name  string
	start uint64
	end   uint64
	size  uint64
}

func getsym(name string) (syms []sym, err error) {
	f, err := os.Open(name)
	if err != nil {
		return
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		var y sym

		line := s.Text()
		n, err := fmt.Sscanf(line, "%v %c %s", &y.start, &y.typ, &y.name)
		if n != 3 || err != nil {
			continue
		}
		syms = append(syms, y)
	}

	sort.Slice(syms, func(i, j int) bool {
		return syms[i].start < syms[j].start
	})

	for i := 0; i < len(syms)-1; i++ {
		syms[i].end = syms[i].start
		for j := i + 1; j < len(syms)-1; j++ {
			if syms[j].start > syms[i].start {
				syms[i].end = syms[j].start - 1
				break
			}
		}
		syms[i].size = syms[i].end - syms[i].start + 1
	}

	return
}

func dump(r io.Reader, syms []sym) {
	s := bufio.NewScanner(r)
	o := binary.ByteOrder(binary.LittleEndian)
	if *end != "little" {
		o = binary.BigEndian
	}

loop:
	for {
		var (
			addr uint64
			err  error
		)

		if *bin {
			err = binary.Read(r, o, &addr)
		} else {
			s.Scan()
			_, err = fmt.Sscanf(s.Text(), "%v", &addr)
			if err != io.EOF && err != nil {
				fmt.Fprintf(os.Stderr, "%q: %v\n", s.Text(), err)
				continue
			}
		}
		if err == io.EOF {
			break
		}

		for _, s := range syms {
			if s.start <= addr && addr <= s.end {
				fmt.Printf("%#016x: %-42s %#016x-%#016x %c\n", addr, s.name, s.start, s.end, s.typ)
				continue loop
			}
		}
		fmt.Printf("%#016x: %-36s\n", addr, "no match")
	}
}
