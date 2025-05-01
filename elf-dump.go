package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/qeedquan/go-media/debug/elfutil"
)

var (
	Aflag = flag.Bool("a", false, "dump all sections")
	Dflag = flag.Bool("d", false, "dump dynamic symbols")
	Iflag = flag.Bool("i", false, "dump imported libraries and symbols")
	Sflag = flag.Bool("s", false, "dump symbol table")

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

	switch {
	case *Dflag:
		fmt.Println("Dynamic symbols:")
		dyn, err := f.DynamicSymbols()
		if err != nil {
			warnf("Failed to get dynamic symbols: %v", err)
		}
		for _, s := range dyn {
			fmt.Printf("%s %s %s\n", s.Name, s.Version, s.Library)
		}

	case *Iflag:
		fmt.Println("Imported Libraries:")
		lib, err := f.ImportedLibraries()
		if err != nil {
			warnf("Failed to get imported libraries: %v", err)
		}
		for i := range lib {
			fmt.Println(lib[i])
		}
		fmt.Println()

		fmt.Println("Imported symbols: ")
		sym, err := f.ImportedSymbols()
		if err != nil {
			warnf("Failed to get imported symbols: %v", err)
		}
		for _, s := range sym {
			fmt.Printf("%s %s %s\n", s.Name, s.Version, s.Library)
		}

	case *Sflag:
		sym, err := f.Symbols()
		if err != nil {
			warnf("Failed to get symbols: %v", err)
		}

		fmt.Println("Symbols: ")
		for _, s := range sym {
			fmt.Printf("%s %s %s\n", s.Name, s.Version, s.Library)
		}

	case *Aflag:
		base := filepath.Base(name)
		dir := fmt.Sprintf("%s_sections", base)
		dumpallsects(f, dir)

	default:
		usage()
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
		err := os.WriteFile(name, p.Data, 0644)
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
		err := os.WriteFile(name, s.Data, 0644)
		if err != nil {
			warnf("failed to write section %s: %v", s.Name, err)
		}
	}
}
