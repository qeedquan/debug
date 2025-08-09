package main

import (
	"debug/plan9obj"
	"flag"
	"fmt"
	"os"
)

var (
	status = 0
)

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
	}

	for _, name := range flag.Args() {
		ek(readaout(name))
	}
	os.Exit(status)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: readaout [options] file ...")
	flag.PrintDefaults()
	os.Exit(2)
}

func ek(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "readaout:", err)
		status = 1
	}
}

func readaout(name string) error {
	f, err := plan9obj.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	syms, _ := f.Symbols()

	if flag.NArg() > 1 {
		fmt.Printf("\nFile: %s\n", name)
	}

	mach := "UNKNOWN"
	switch f.Magic {
	case plan9obj.Magic386:
		mach = "386"
	case plan9obj.MagicAMD64:
		mach = "AMD64"
	case plan9obj.MagicARM:
		mach = "ARM"
	}

	fmt.Printf("AOUT Header:\n")
	fmt.Printf("  Magic:               %02x %02x %02x %02x\n",
		byte(f.Magic>>24), byte(f.Magic>>16), byte(f.Magic>>8), byte(f.Magic))
	fmt.Printf("  Architecture:        %s\n", mach)
	fmt.Printf("  Entry Point Address: %#x\n", f.Entry)
	fmt.Printf("  Load Address:        %#x\n", f.LoadAddress)
	fmt.Printf("  Number of Sections:  %d\n", len(f.Sections))
	fmt.Printf("  Number of Symbols:   %d\n", len(syms))
	fmt.Printf("  Pointer Size:        %d\n", f.PtrSize)
	fmt.Printf("  BSS Size:            %d\n", f.Bss)
	fmt.Printf("\n\n")

	fmt.Printf("Section Headers:\n")
	fmt.Printf("  [Nr] Name    Size        Offset\n")
	for i, s := range f.Sections {
		fmt.Printf("  [%d]  %s    %-8d    %#-8x\n", i, s.Name, s.Size, s.Offset)
	}
	fmt.Printf("\n\n")

	fmt.Printf("Symbols:\n")
	for _, s := range syms {
		fmt.Printf("    %c %#08x %q\n", s.Type, s.Value, s.Name)
	}

	return nil
}
