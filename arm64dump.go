package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	"golang.org/x/arch/arm64/arm64asm"
)

var (
	offset = flag.Int("o", 0, "decode at offset")
	size   = flag.Int("s", -1, "read up to size")
	syntax = flag.String("y", "gnu", "syntax [gnu | go]")

	bout = bufio.NewWriter(os.Stdout)

	status = 0
)

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		ek(dis("<stdin>", os.Stdin))
	} else {
		for _, name := range flag.Args() {
			f, err := os.Open(name)
			if ek(err) {
				continue
			}
			ek(dis(name, f))
			f.Close()
		}
	}
	bout.Flush()
	os.Exit(status)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: arm64dump [options] [file ...]")
	flag.PrintDefaults()
	os.Exit(2)
}

func ek(err error) bool {
	if err != nil {
		bout.Flush()
		fmt.Fprintln(os.Stderr, "arm64dump:", err)
		status = 1
		return true
	}
	return false
}

func dis(name string, r io.Reader) error {
	var buf []byte
	var err error

	if *size >= 0 {
		r = &io.LimitedReader{r, int64(*size)}
	}
	buf, err = io.ReadAll(r)

	if err != nil {
		return err
	}
	if int(len(buf)) < *offset {
		return fmt.Errorf("%v: invalid offset", name)
	}
	buf = buf[*offset:]

	pos := *offset
	for len(buf) > 4 {
		var loc string
		if flag.NArg() < 2 {
			loc = fmt.Sprintf("%x:", pos)
		} else {
			loc = fmt.Sprintf("%v:%x:", name, pos)
		}

		inst, err := arm64asm.Decode(buf)
		if err != nil {
			fmt.Fprintf(bout, "%-8s %08x     %s\n", loc, buf[0], err)
			nxt := 4
			buf = buf[nxt:]
			pos += nxt
			continue
		}

		var op string
		switch *syntax {
		case "gnu":
			op = arm64asm.GNUSyntax(inst)
		case "go":
			op = inst.String()
		default:
			return fmt.Errorf("unknown syntax %q", *syntax)
		}
		fmt.Fprintf(bout, "%-8s %08x     %s\n", loc, inst.Enc, op)

		nxt := 4
		buf = buf[nxt:]
		pos += nxt
	}
	return nil
}
