package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/qeedquan/go-media/debug/extekhex"
	"github.com/qeedquan/go-media/debug/srec"
	"github.com/qeedquan/go-media/debug/ti/tagged"
)

var (
	from   = flag.String("f", "mt3", "source format")
	to     = flag.String("t", "bin", "destination format")
	output = flag.String("o", "", "output file")

	status = 0
)

func main() {
	log.SetPrefix("tihex: ")
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		err := conv(os.Stdin, os.Stdout, *from, *to)
		chkerr("<stdin>", err)
	} else {
		name := flag.Arg(0)
		r, err := os.Open(name)
		if chkerr("", err) {
			goto out
		}
		defer r.Close()

		if *output == "" {
			*output = mkoutname(name, *to)
		}
		w, err := os.Create(*output)
		if chkerr("", err) {
			goto out
		}
		defer w.Close()

		chkerr(name, conv(r, w, *from, *to))
	}

out:
	os.Exit(status)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: [options] file")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "\navailable formats: ")
	fmt.Fprintln(os.Stderr, "bin, extekhex, mt1, mt2, mt3, titag")
	os.Exit(2)
}

func chkerr(name string, err error) bool {
	if err != nil {
		if name != "" {
			err = fmt.Errorf("%v: %v", name, err)
		}
		fmt.Fprintln(os.Stderr, err)
		status = 1
		return true
	}
	return false
}

func conv(r io.Reader, w io.Writer, from, to string) (err error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return
	}

	switch from {
	case "mt1", "mt2", "mt3":
		src, err = mt2bin(src)
	case "titag":
		src, err = tagged2bin(src)
	case "extekhex":
		src, err = extekhex2bin(src)
	case "bin":
	default:
		err = fmt.Errorf("unsupported input format %q", from)
	}
	if err != nil {
		return
	}

	var dst []byte
	switch to {
	case "bin":
		dst = src
	default:
		err = fmt.Errorf("unsupported output format %q", to)
	}
	if err != nil {
		return
	}

	_, err = w.Write(dst)
	if err != nil {
		return
	}

	return
}

func mkoutname(name, format string) string {
	ext := filepath.Ext(name)
	out := name[:len(name)-len(ext)]
	oext := ".out"
	switch format {
	case "mt1", "mt2", "mt3", "titag", "extekhex":
		oext = "." + format
	case "bin":
		oext = ".bin"
	}

	out += oext
	if out == name {
		out += oext
	}

	return out
}

func mt2bin(src []byte) (dst []byte, err error) {
	f, err := srec.Decode(bytes.NewReader(src))
	if err != nil {
		return
	}
	dst = f.Binary()
	return
}

func tagged2bin(src []byte) (dst []byte, err error) {
	f, err := tagged.Decode(bytes.NewReader(src))
	if err != nil {
		return
	}
	dst = f.Binary()
	return
}

func extekhex2bin(src []byte) (dst []byte, err error) {
	f, err := extekhex.Decode(bytes.NewReader(src))
	if err != nil {
		return
	}
	dst = f.Binary()
	return
}
