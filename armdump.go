package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	"golang.org/x/arch/arm/armasm"
)

var (
	offset = flag.Int("o", 0, "decode at offset")
	mode   = flag.String("m", "arm", "instruction mode [arm | thumb]")
	size   = flag.Int("s", -1, "read up to size")
	syntax = flag.String("y", "gnu", "syntax [gnu | go]")
	fix    = flag.Int("f", 4, "fixed length instruction")

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
	fmt.Fprintln(os.Stderr, "usage: armdump [options] [file ...]")
	flag.PrintDefaults()
	os.Exit(2)
}

func ek(err error) bool {
	if err != nil {
		bout.Flush()
		fmt.Fprintln(os.Stderr, "armdump:", err)
		status = 1
		return true
	}
	return false
}

func ilen(n int) int {
	if *fix != 0 {
		return *fix
	}
	return n
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

	var mod armasm.Mode
	switch *mode {
	case "arm":
		mod = armasm.ModeARM
	case "thumb":
		mod = armasm.ModeThumb
	default:
		return fmt.Errorf("invalid mode %q", *mode)
	}

	pos := *offset
	for len(buf) > 0 {
		var loc string
		if flag.NArg() < 2 {
			loc = fmt.Sprintf("%x:", pos)
		} else {
			loc = fmt.Sprintf("%v:%x:", name, pos)
		}

		inst, err := armasm.Decode(buf, mod)
		if err != nil {
			fmt.Fprintf(bout, "%-8s %08x     %s\n", loc, buf[0], err)
			nxt := ilen(1)
			if nxt >= len(buf) {
				break
			}
			buf = buf[nxt:]
			pos += nxt
			continue
		}

		var op string
		switch *syntax {
		case "gnu":
			op = armasm.GNUSyntax(inst)
		case "go":
			op = inst.String()
		default:
			return fmt.Errorf("unknown syntax %q", *syntax)
		}
		fmt.Fprintf(bout, "%-8s %08x     %s\n", loc, inst.Enc, op)

		nxt := ilen(inst.Len)
		buf = buf[nxt:]
		pos += nxt
	}
	return nil
}
