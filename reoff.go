package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func main() {
	var p Reoff
	flag.Uint64Var(&p.base, "base", 0, "base offset")
	flag.Uint64Var(&p.reloc, "reloc", 0, "relocation offset")
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 {
		p.Run(os.Stdin)
	} else {
		for _, arg := range flag.Args() {
			f, err := os.Open(arg)
			if err != nil {
				p.Process(arg)
			} else {
				p.Run(f)
				f.Close()
			}
		}
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: [options] <input> ...")
	flag.PrintDefaults()
	os.Exit(2)
}

type Reoff struct {
	base  uint64
	reloc uint64
}

func (p *Reoff) Run(r io.Reader) {
	scan := bufio.NewScanner(r)
	for scan.Scan() {
		p.Process(scan.Text())
	}
}

func (p *Reoff) Process(line string) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "#") || line == "" {
		fmt.Printf("%s\n", line)
		return
	}

	i := strings.Index(line, "#")
	if i >= 0 {
		line = line[:i]
	}

	toks := strings.Split(line, " ")
	if len(toks) > 2 {
		return
	}

	var (
		val uint64
		err error
	)
	if len(toks) == 1 {
		val, err = strconv.ParseUint(toks[0], 0, 64)
	} else {
		val, err = strconv.ParseUint(toks[1], 0, 64)
	}

	if err != nil {
		fmt.Printf("%s -> failed to reoffset\n", line)
		return
	}

	switch strings.ToLower(toks[0]) {
	case "base":
		p.base = val
		fmt.Printf("base = %#x\n", p.base)
	case "reloc":
		p.reloc = val
		fmt.Printf("reloc = %#x\n", p.reloc)
	default:
		fmt.Printf("%#x -> %#x\n", val, val-p.base+p.reloc)
	}
}
