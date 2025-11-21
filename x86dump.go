package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"golang.org/x/arch/x86/x86asm"
)

var (
	offset = flag.Int("o", 0, "decode at offset")
	mode   = flag.Int("m", 64, "instruction mode")
	size   = flag.Int("s", -1, "read up to size")
	syntax = flag.String("y", "intel", "syntax [intel | gnu | go]")

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
	fmt.Fprintln(os.Stderr, "usage: x86dump [options] [file ...]")
	flag.PrintDefaults()
	os.Exit(2)
}

func ek(err error) bool {
	if err != nil {
		bout.Flush()
		fmt.Fprintln(os.Stderr, "x86dump:", err)
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

	pos := uint64(*offset)
	for len(buf) > 0 {
		var loc string
		if flag.NArg() < 2 {
			loc = fmt.Sprintf("%x:", pos)
		} else {
			loc = fmt.Sprintf("%v:%x:", name, pos)
		}

		inst, err := x86asm.Decode(buf, *mode)
		if err != nil {
			fmt.Fprintf(bout, "%-8s %-32x %s\n", loc, buf[0], err)
			buf = buf[1:]
			pos++
			continue
		}

		var op string
		switch *syntax {
		case "intel":
			op = x86asm.IntelSyntax(inst, pos, nil)
		case "gnu":
			op = x86asm.GNUSyntax(inst, pos, nil)
		case "go":
			op = inst.String()
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
		pos += uint64(inst.Len)
	}
	return nil
}
