package main

import (
	"debug/pe"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/qeedquan/go-media/debug/peutil"
)

var (
	dflag = flag.Bool("d", false, "dump dwarf")
	sflag = flag.Bool("s", false, "dump strings")
	vflag = flag.Bool("v", false, "dump everything")
	yflag = flag.Bool("y", false, "dump debug symbols")
	rflag = flag.String("r", "", "dump all section data into directory")
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("pe-dump: ")

	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 1 {
		usage()
	}
	if *vflag {
		*yflag = true
		*sflag = true
		*dflag = true
	}

	dump(flag.Arg(0))
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: pe-dump [options] file")
	flag.PrintDefaults()
	os.Exit(2)
}

func ck(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func dump(name string) {
	f, err := peutil.Open(name)
	ck(err)
	defer f.Close()

	if *rflag != "" {
		dumpSectionData(f, *rflag)
		return
	}

	var (
		dllch    uint32
		imgbase  uint64
		imgsize  uint64
		entry    uint64
		codebase uint64
		database uint64
	)
	switch h := f.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		dllch = uint32(h.DllCharacteristics)
		imgbase = uint64(h.ImageBase)
		imgsize = uint64(h.SizeOfImage)
		entry = uint64(h.AddressOfEntryPoint) + imgbase
		codebase = uint64(h.BaseOfCode) + imgbase
		database = uint64(h.BaseOfData) + imgbase
	case *pe.OptionalHeader64:
		dllch = uint32(h.DllCharacteristics)
		imgbase = uint64(h.ImageBase)
		imgsize = uint64(h.SizeOfImage)
		entry = uint64(h.AddressOfEntryPoint) + imgbase
		codebase = uint64(h.BaseOfCode) + imgbase
	default:
		imgbase = f.ImageBase
	}

	var entryname, codename, dataname string
	entrysect, _, entryoff := f.LookupVirtualAddress(entry - imgbase)
	codesect, _, codeoff := f.LookupVirtualAddress(codebase - imgbase)
	datasect, _, dataoff := f.LookupVirtualAddress(database - imgbase)
	if entrysect != nil {
		entryname = entrysect.Name
	}
	if codesect != nil {
		codename = codesect.Name
	}
	if datasect != nil {
		dataname = datasect.Name
	}
	fmt.Printf("Machine Type         : %s\n", peutil.MachineType(f.Machine))
	fmt.Printf("Image Base           : %#x\n", imgbase)
	fmt.Printf("Entry                : %#x (%s: %#x)\n", entry, entryname, entryoff)
	fmt.Printf("Code Base            : %#x (%s: %#x)\n", codebase, codename, codeoff)
	fmt.Printf("Data Base            : %#x (%s: %#x)\n", database, dataname, dataoff)
	fmt.Printf("Number of Sections   : %d\n", len(f.Sections))
	fmt.Printf("Pointer Symbol Table : %#x (%d)\n", f.PointerToSymbolTable, f.PointerToSymbolTable)
	fmt.Printf("Size Of Image        : %#x (%d)\n", imgsize, imgsize)
	fmt.Printf("File Alignment       : %#x (%d)\n", f.FileAlignment, f.FileAlignment)
	fmt.Printf("Section Alignment    : %#x (%d)\n", f.SectionAlignment, f.SectionAlignment)
	fmt.Printf("Optional Header Size : %d\n", f.SizeOfOptionalHeader)
	fmt.Println()

	if dllch != 0 {
		chars := []struct {
			bit uint32
			str string
		}{
			{peutil.IMAGE_DLLCHARACTERISTICS_DYNAMIC_BASE, "IMAGE_DLLCHARACTERISTICS_DYNAMIC_BASE"},
			{peutil.IMAGE_DLLCHARACTERISTICS_FORCE_INTEGRITY, "IMAGE_DLLCHARACTERISTICS_FORCE_INTEGRITY"},
			{peutil.IMAGE_DLLCHARACTERISTICS_NX_COMPAT, "IMAGE_DLLCHARACTERISTICS_NX_COMPAT"},
			{peutil.IMAGE_DLLCHARACTERISTICS_NO_ISOLATION, "IMAGE_DLLCHARACTERISTICS_NO_ISOLATION"},
			{peutil.IMAGE_DLLCHARACTERISTICS_NO_SEH, "IMAGE_DLLCHARACTERISTICS_NO_SEH"},
			{peutil.IMAGE_DLLCHARACTERISTICS_NO_BIND, "IMAGE_DLLCHARACTERISTICS_NO_BIND"},
			{peutil.IMAGE_DLLCHARACTERISTICS_WDM_DRIVER, "IMAGE_DLLCHARACTERISTICS_WDM_DRIVER"},
			{peutil.IMAGE_DLLCHARACTERISTICS_TERMINAL_SERVER_AWARE, "IMAGE_DLLCHARACTERISTICS_TERMINAL_SERVER_AWARE"},
		}

		fmt.Printf("DLL Characteristics: %#x\n", dllch)
		for _, c := range chars {
			if dllch&c.bit != 0 {
				fmt.Println(c.str)
			}
		}
		fmt.Println()
	}

	for _, s := range f.Sections {
		chars := []struct {
			bit uint32
			str string
		}{
			{peutil.IMAGE_SCN_TYPE_NO_PAD, "IMAGE_SCN_TYPE_NO_PAD"},
			{peutil.IMAGE_SCN_CNT_CODE, "IMAGE_SCN_CNT_CODE"},
			{peutil.IMAGE_SCN_CNT_INITIALIZED_DATA, "IMAGE_SCN_CNT_INITIALIZED_DATA"},
			{peutil.IMAGE_SCN_CNT_UNINITIALIZED_DATA, "IMAGE_SCN_CNT_UNINITIALIZED_DATA"},
			{peutil.IMAGE_SCN_LNK_OTHER, "IMAGE_SCN_LNK_OTHER"},
			{peutil.IMAGE_SCN_LNK_INFO, "IMAGE_SCN_LNK_INFO"},
			{peutil.IMAGE_SCN_LNK_REMOVE, "IMAGE_SCN_LNK_REMOVE"},
			{peutil.IMAGE_SCN_LNK_COMDAT, "IMAGE_SCN_LNK_COMDAT"},
			{peutil.IMAGE_SCN_GPREL, "IMAGE_SCN_GPREL"},
			{peutil.IMAGE_SCN_MEM_PURGEABLE, "IMAGE_SCN_MEM_PURGEABLE"},
			{peutil.IMAGE_SCN_MEM_16BIT, "IMAGE_SCN_MEM_16BIT"},
			{peutil.IMAGE_SCN_MEM_LOCKED, "IMAGE_SCN_MEM_LOCKED"},
			{peutil.IMAGE_SCN_MEM_PRELOAD, "IMAGE_SCN_MEM_PRELOAD"},
			{peutil.IMAGE_SCN_ALIGN_1BYTES, "IMAGE_SCN_ALIGN_1BYTES"},
			{peutil.IMAGE_SCN_ALIGN_2BYTES, "IMAGE_SCN_ALIGN_2BYTES"},
			{peutil.IMAGE_SCN_ALIGN_8BYTES, "IMAGE_SCN_ALIGN_8BYTES"},
			{peutil.IMAGE_SCN_ALIGN_16BYTES, "IMAGE_SCN_ALIGN_16BYTES"},
			{peutil.IMAGE_SCN_ALIGN_32BYTES, "IMAGE_SCN_ALIGN_32BYTES"},
			{peutil.IMAGE_SCN_ALIGN_64BYTES, "IMAGE_SCN_ALIGN_64BYTES"},
			{peutil.IMAGE_SCN_ALIGN_128BYTES, "IMAGE_SCN_ALIGN_128BYTES"},
			{peutil.IMAGE_SCN_ALIGN_256BYTES, "IMAGE_SCN_ALIGN_256BYTES"},
			{peutil.IMAGE_SCN_ALIGN_512BYTES, "IMAGE_SCN_ALIGN_512BYTES"},
			{peutil.IMAGE_SCN_ALIGN_1024BYTES, "IMAGE_SCN_ALIGN_1024BYTES"},
			{peutil.IMAGE_SCN_ALIGN_2048BYTES, "IMAGE_SCN_ALIGN_2048BYTES"},
			{peutil.IMAGE_SCN_ALIGN_4096BYTES, "IMAGE_SCN_ALIGN_4096BYTES"},
			{peutil.IMAGE_SCN_ALIGN_8192BYTES, "IMAGE_SCN_ALIGN_8192BYTES"},
			{peutil.IMAGE_SCN_LNK_NRELOC_OVFL, "IMAGE_SCN_LNK_NRELOC_OVFL"},
			{peutil.IMAGE_SCN_MEM_DISCARDABLE, "IMAGE_SCN_MEM_DISCARDABLE"},
			{peutil.IMAGE_SCN_MEM_NOT_CACHED, "IMAGE_SCN_MEM_NOT_CACHED"},
			{peutil.IMAGE_SCN_MEM_NOT_PAGED, "IMAGE_SCN_MEM_NOT_PAGED"},
			{peutil.IMAGE_SCN_MEM_SHARED, "IMAGE_SCN_MEM_SHARED"},
			{peutil.IMAGE_SCN_MEM_EXECUTE, "IMAGE_SCN_MEM_EXECUTE"},
			{peutil.IMAGE_SCN_MEM_READ, "IMAGE_SCN_MEM_READ"},
			{peutil.IMAGE_SCN_MEM_WRITE, "IMAGE_SCN_MEM_WRITE"},
		}

		fmt.Printf("Section %s\n", s.Name)
		fmt.Printf("Virtual Address          : %#x - %#x\n", imgbase+uint64(s.VirtualAddress), imgbase+uint64(s.VirtualAddress)+uint64(s.VirtualSize))
		fmt.Printf("Virtual Address (Raw)    : %#x - %#x\n", s.VirtualAddress, s.VirtualAddress+s.VirtualSize)
		fmt.Printf("Virtual Size             : %#x (%d bytes)\n", s.VirtualSize, s.VirtualSize)
		fmt.Printf("Size of Raw Data         : %#x (%d bytes)\n", s.Size, s.Size)
		fmt.Printf("Pointer to Raw Data      : %#x - %#x\n", s.Offset, s.Offset+s.Size)
		fmt.Printf("Pointer To Relocations   : %#x (%d)\n", s.PointerToRelocations, s.PointerToRelocations)
		fmt.Printf("Pointer To Line Numbers  : %#x (%d)\n", s.PointerToLineNumbers, s.PointerToLineNumbers)
		fmt.Printf("Number of Relocations    : %d\n", s.NumberOfRelocations)
		fmt.Printf("Number of Line Numbers   : %d\n", s.NumberOfLineNumbers)
		fmt.Printf("Characteristics          : %#x\n", s.Characteristics)
		for _, c := range chars {
			if s.Characteristics&c.bit != 0 {
				fmt.Println(c.str)
			}
		}
		fmt.Println()
	}

	dirNames := []string{
		"IMAGE_DIRECTORY_ENTRY_EXPORT",
		"IMAGE_DIRECTORY_ENTRY_IMPORT",
		"IMAGE_DIRECTORY_ENTRY_RESOURCE",
		"IMAGE_DIRECTORY_ENTRY_EXCEPTION",
		"IMAGE_DIRECTORY_ENTRY_SECURITY",
		"IMAGE_DIRECTORY_ENTRY_BASERELOC",
		"IMAGE_DIRECTORY_ENTRY_DEBUG",
		"IMAGE_DIRECTORY_ENTRY_ARCHITECTURE",
		"IMAGE_DIRECTORY_ENTRY_GLOBALPTR",
		"IMAGE_DIRECTORY_ENTRY_TLS",
		"IMAGE_DIRECTORY_ENTRY_LOAD_CONFIG",
		"IMAGE_DIRECTORY_ENTRY_BOUND_IMPORT",
		"IMAGE_DIRECTORY_ENTRY_IAT",
		"IMAGE_DIRECTORY_ENTRY_DELAY_IMPORT",
		"IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR",
	}
	for i, name := range dirNames {
		d := f.DataDirectory(i)
		if d == nil {
			break
		}
		s, _, off := f.LookupVirtualAddress(uint64(d.VirtualAddress))

		fmt.Printf("Data Directory %s\n", name)
		if s != nil {
			fmt.Printf("Section               : %s\n", s.Name)
			fmt.Printf("Section Offset        : %#x - %#x\n", off, off+int(d.Size))
		}
		if d.Size > 0 {
			fmt.Printf("Virtual Address       : %#x - %#x\n", imgbase+uint64(d.VirtualAddress), imgbase+uint64(d.VirtualAddress+d.Size))
		}
		fmt.Printf("Virtual Address (Raw) : %#x - %#x\n", uint64(d.VirtualAddress), uint64(d.VirtualAddress+d.Size))
		fmt.Printf("Size                  : %#x (%d bytes)\n", d.Size, d.Size)
		fmt.Printf("\n")
	}

	spew.Dump(f.FileHeader)
	fmt.Println()
	spew.Dump(f.OptionalHeader)
	fmt.Println("\n")

	if *yflag {
		fmt.Println("Symbols:")
		spew.Dump(f.Symbols)
		fmt.Println()

		fmt.Println("Coff Symbols:")
		spew.Dump(f.COFFSymbols)
		fmt.Println()

		fmt.Println("String Table:")
		spew.Dump(f.StringTable)
		fmt.Println()
	}

	if *dflag {
		dumpDWARF(f)
	}

	il, _ := f.ImportedLibraries()
	fmt.Println("Imported Libraries:")
	spew.Dump(il)
	fmt.Println()

	if f.OptionalHeader != nil {
		is, err := f.ReadImportTable()
		nsym := 0
		for _, d := range is {
			nsym += len(d.Symbols)
		}
		fmt.Printf("Imported Symbols: (%d symbols)\n", nsym)
		fmt.Println(strings.Repeat("-", 80))
		if err == nil {
			for _, d := range is {
				for _, y := range d.Symbols {
					var s [4]*peutil.Section
					var a [4]string
					s[0], _, _ = f.LookupVirtualAddress(y.DLLNameRVA)
					s[1], _, _ = f.LookupVirtualAddress(y.NameRVA)
					s[2], _, _ = f.LookupVirtualAddress(y.OriginalThunkRVA)
					s[3], _, _ = f.LookupVirtualAddress(y.ThunkRVA)
					for i := range a {
						if s[i] != nil {
							a[i] = s[i].Name
						}
					}
					fmt.Printf("%-16s %-70s %#08x %s %#08x %s %#08x %s %#08x %s",
						d.DLLName, y.Name, f.ImageBase+y.DLLNameRVA, a[0], f.ImageBase+y.NameRVA, a[1], f.ImageBase+y.OriginalThunkRVA, a[2], f.ImageBase+y.ThunkRVA, a[3])
					fmt.Println()
				}
			}
		}
		fmt.Println()

		sym, _ := f.ExportedSymbols()
		numpad := 1
		if len(sym) > 0 {
			numpad = int(math.Log10(float64(len(sym)))) + 1
		}

		fmt.Printf("Exported Symbols: (%d symbols)\n", len(sym))
		fmt.Println(strings.Repeat("-", 80))
		for i, y := range sym {
			s, _, _ := f.LookupVirtualAddress(y.NameRVA)
			a := ""
			if s != nil {
				a = s.Name
			}
			fmt.Printf("%*d %-80s %#08x %s\n", numpad, i+1, y.Name, f.ImageBase+y.NameRVA, a)
		}
		fmt.Println()
	}

	if *sflag {
		fmt.Println("Strings: ")
		stab := f.FindStrings()
		for _, st := range stab {
			fmt.Println(st)
		}
	}
}

func dumpDWARF(f *peutil.File) {
	dw, err := f.DWARF()
	if err != nil {
		fmt.Printf("No DWARF Info\n\n")
	} else {
		fmt.Println("DWARF Info:")
		spew.Dump(dw)
	}
}

func dumpSectionData(f *peutil.File, dir string) {
	for _, s := range f.Sections {
		name := s.Name
		name = strings.Replace(name, ".", "_", -1)
		name = "section" + name
		name = filepath.Join(dir, name)

		xdir := filepath.Dir(name)
		os.MkdirAll(xdir, 0755)

		err := os.WriteFile(name, s.Data, 0644)
		if err == nil {
			fmt.Println("Wrote out section", name)
		} else {
			fmt.Println(name, err)
		}
	}
}
