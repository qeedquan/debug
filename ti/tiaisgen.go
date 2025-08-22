package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/qeedquan/go-media/debug/ti/ais"
)

var (
	outfile = flag.String("o", "ais_output.bin", "output file")
	entry   = flag.Int64("e", -1, "entry point")
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("tiaisgen: ")

	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
	}

	m := new(ais.Image)
	ck(addini(m, flag.Arg(0)))
	for i := 1; i < flag.NArg(); i++ {
		ck(addfile(m, flag.Arg(i)))
	}
	if *entry >= 0 {
		m.Cmds = append(m.Cmds, ais.Cmd{Op: ais.JUMP_CLOSE, Addr: uint32(*entry)})
	}
	ck(writeimg(m, *outfile))
}

func ck(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: [options] <ini> [file] ...")
	flag.PrintDefaults()
	os.Exit(2)
}

func addini(m *ais.Image, name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return m.AddINI(f)
}

func writeimg(m *ais.Image, name string) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}

	err = ais.Format(m, f)
	xerr := f.Close()
	if err == nil {
		err = xerr
	}
	return err
}

func addfile(m *ais.Image, name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}

	e, err := m.AddELF(f)
	if err == nil {
		*entry = int64(e.Entry)
		return nil
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	m.Cmds = append(m.Cmds, ais.Cmd{Op: ais.SECTION_LOAD, Addr: 0, Data: b})
	return nil
}
