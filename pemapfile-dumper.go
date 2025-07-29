package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"golang.org/x/arch/arm/armasm"
	"golang.org/x/arch/arm64/arm64asm"
	"golang.org/x/arch/x86/x86asm"

	"github.com/qeedquan/go-media/debug/pemapfile"
	"github.com/qeedquan/go-media/debug/peutil"
)

var flags struct {
	OutputDir   string
	Pattern     string
	Arch        string
	NoDump      bool
	ShowCallers bool
	BaseAddr    uint64
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("pemapfile-dumper: ")

	flag.StringVar(&flags.Arch, "a", runtime.GOARCH, "architecture for assembly analyis features")
	flag.BoolVar(&flags.ShowCallers, "c", false, "show callers")
	flag.BoolVar(&flags.NoDump, "nd", false, "don't dump but print the processing output")
	flag.StringVar(&flags.OutputDir, "o", ".", "output directory")
	flag.StringVar(&flags.Pattern, "p", ".*", "symbol pattern to match")
	flag.Uint64Var(&flags.BaseAddr, "b", 0, "custom base address")
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 2 {
		usage()
	}

	pf, err := peutil.Open(flag.Arg(0))
	ck(err)

	mf, err := pemapfile.Open(flag.Arg(1))
	ck(err)

	err = dump(pf, mf, flags.OutputDir, flags.Pattern)
	ck(err)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: [options] <exefile> <mapfile>")
	flag.PrintDefaults()
	os.Exit(2)
}

func ck(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func dump(pf *peutil.File, mf *pemapfile.File, outdir, pat string) error {
	if !flags.NoDump {
		os.MkdirAll(outdir, 0755)
	}

	re, err := regexp.Compile(pat)
	if err != nil {
		return err
	}

	w := os.Stdout
	for i := 0; i < len(mf.Symbols); i++ {
		y := &mf.Symbols[i]
		if !re.MatchString(y.Name) {
			continue
		}

		ms := &mf.Sections[y.Section]
		ps := findsect(pf, ms.Name)
		if ps == nil {
			continue
		}
		if y.Offset >= uint64(len(ps.Data)) || y.Offset+y.Size >= uint64(len(ps.Data)) {
			continue
		}

		name := filepath.Join(outdir, y.Name)
		data := ps.Data[y.Offset : y.Offset+y.Size]
		fmt.Fprintf(w, "%-90q %#016x-%#016x %#016x-%#016x %#08x-%#08x %#08x\n",
			y.Name,
			flags.BaseAddr+y.Offset, flags.BaseAddr+y.Offset+y.Size,
			y.Addr, y.Addr+y.Size,
			y.Offset, y.Offset+y.Size, y.Size)

		if flags.ShowCallers {
			callers(mf, y, data, w)
		}

		if flags.NoDump {
			continue
		}

		err = os.WriteFile(name, data, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to write symbol %v: %v\n", y.Name, err)
		}
	}

	return nil
}

func findsect(pf *peutil.File, sect string) *peutil.Section {
	for i := range pf.Sections {
		p := pf.Sections[i]
		if strings.HasPrefix(sect, p.Name) {
			return p
		}
	}
	return nil
}

func callers(mf *pemapfile.File, sym *pemapfile.Symbol, data []byte, w io.Writer) {
	pc := 0
	for len(data) > 0 {
		var (
			rel    int64
			length int
			err    error
		)

		switch flags.Arch {
		case "amd64", "386":
			bits := 64
			if flags.Arch == "386" {
				bits = 32
			}
			var inst x86asm.Inst
			inst, err = x86asm.Decode(data, bits)
			length = inst.Len
			if inst.Op == x86asm.CALL {
				switch arg := inst.Args[0].(type) {
				case x86asm.Rel:
					rel = int64(arg)
				}
			}

		case "arm":
			var inst armasm.Inst
			inst, err = armasm.Decode(data, armasm.ModeARM)
			length = inst.Len

		case "arm64":
			_, err = arm64asm.Decode(data)
			length = 4
		}
		if err != nil {
			length = 1
		}
		data = data[length:]
		pc += length

		if rel != 0 {
			symoff := int64(sym.Offset) + int64(pc)
			off := uint64(symoff + rel)
			for _, y := range mf.Symbols {
				if y.Segment != sym.Segment {
					continue
				}

				if y.Offset <= off && off < y.Offset+y.Size {
					fmt.Fprintf(w, "    %#016x %#016x %#016x %-90q %#016x-%#016x %#016x-%#016x %#08x-%#08x %#08x\n",
						int64(flags.BaseAddr)+symoff, symoff,
						off, y.Name,
						flags.BaseAddr+y.Offset, flags.BaseAddr+y.Offset+y.Size,
						y.Addr, y.Addr+y.Size,
						y.Offset, y.Offset+y.Size, y.Size)
					break
				}
			}
		}

	}
	fmt.Fprintf(w, "\n")
}
