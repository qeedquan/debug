package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"

	"golang.org/x/arch/ppc64/ppc64asm"
)

var (
	offset = flag.Int("o", 0, "decode at offset")
	size   = flag.Int("s", -1, "read up to size")
	syntax = flag.String("y", "gnu", "syntax")

	bout = bufio.NewWriter(os.Stdout)

	order binary.ByteOrder = binary.BigEndian

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
	fmt.Fprintln(os.Stderr, "usage: ppcdump [options] [file ...]")
	flag.PrintDefaults()
	os.Exit(2)
}

func ek(err error) bool {
	if err != nil {
		bout.Flush()
		fmt.Fprintln(os.Stderr, "ppcdump:", err)
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
	for len(buf) > 0 {
		var loc string
		if flag.NArg() < 2 {
			loc = fmt.Sprintf("%x:", pos)
		} else {
			loc = fmt.Sprintf("%v:%x:", name, pos)
		}

		inst, err := ppc64asm.Decode(buf, order)
		if err != nil {
			fmt.Fprintf(bout, "%-8s %-32x %s\n", loc, buf[0], err)
			buf = buf[1:]
			pos++
			continue
		}

		var op string
		switch *syntax {
		case "gnu":
			op = ppc64asm.GNUSyntax(inst, 0)
		case "go":
			op = ppc64asm.GoSyntax(inst, 0, nil)
		default:
			return fmt.Errorf("unknown syntax %q", *syntax)
		}

		bw := new(bytes.Buffer)
		for i := 0; i < inst.Len; i++ {
			fmt.Fprintf(bw, "%02x ", buf[i])
		}
		opstr := bw.String()
		opstr = opstr[:len(opstr)-1]

		fmt.Fprintf(bout, "%-8s %-32s %s\n", loc, opstr, op)

		buf = buf[inst.Len:]
		pos += inst.Len
	}
	return nil
}
