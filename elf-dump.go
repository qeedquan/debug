package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/qeedquan/go-media/debug/elfutil"
)

var (
	Sflag = flag.Bool("S", false, "dump all sections")

	status = 0
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("elf-dump: ")
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 1 {
		usage()
	}

	name := flag.Arg(0)
	f, err := elfutil.Open(name)
	ck(err)

	if *Sflag {
		base := filepath.Base(name)
		dir := fmt.Sprintf("%s_sections", base)
		dumpallsects(f, dir)
	}
	os.Exit(status)
}

func warnf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	status = 1
}

func ck(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: [options] file")
	flag.PrintDefaults()
	os.Exit(2)
}

func dumpallsects(f *elfutil.File, outdir string) {
	os.MkdirAll(outdir, 0755)

	for i, p := range f.Progs {
		name := fmt.Sprintf("program%d", i)
		fmt.Printf("dumping program %q\n", name)

		name = filepath.Join(outdir, name)
		err := ioutil.WriteFile(name, p.Data, 0644)
		if err != nil {
			warnf("failed to write program section %d: %v", i, err)
		}
	}
	for i, s := range f.Sections {
		name := s.Name
		if name == "" {
			name = fmt.Sprintf("section%d", i)
		}
		fmt.Printf("dumping section %q\n", name)

		name = filepath.Join(outdir, name)
		err := ioutil.WriteFile(name, s.Data, 0644)
		if err != nil {
			warnf("failed to write section %s: %v", s.Name, err)
		}
	}
}
