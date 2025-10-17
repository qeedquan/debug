package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qeedquan/go-media/debug/ti/coff"
)

var (
	dasflag = flag.Bool("das", false, "dump all sections")
)

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
	}

	f, err := coff.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	switch {
	case *dasflag:
		dumpallsect(flag.Arg(0), f)
	default:
		dump(flag.Arg(0), f)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: [options] file")
	flag.PrintDefaults()
	os.Exit(2)
}

func dumpallsect(name string, f *coff.File) {
	dir := fmt.Sprintf("%s_sections", name)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	for _, s := range f.Sections {
		sectname := filepath.Join(dir, s.Name)
		fmt.Printf("dumping section %s offset %#x size %#x\n", s.Name, s.DataOff, s.Size)
		err := os.WriteFile(sectname, s.Data, 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error dumping section %s: %v", s.Name, err)
		}
	}
}

func dump(name string, f *coff.File) {
	dumphdr(name, f)
	dumpsct(f)
	dumpsym(f)
	dumpstr(f)
}

func dumphdr(name string, f *coff.File) {
	var (
		filesz    int64
		entry     uint32
		strtaboff uint32
		textoff   uint32
		dataoff   uint32
		textsz    uint32
		datasz    uint32
		bsssz     uint32
		entrysect string
	)
	fi, err := os.Stat(name)
	if err == nil {
		filesz = fi.Size()
	}
	if h := f.OptionalHeader; h != nil {
		entry = h.Entry
		textoff = h.TextAddr
		dataoff = h.DataAddr
		textsz = h.TextSize
		datasz = h.DataSize
		bsssz = h.BSSSize
	}
	strtaboff = f.SymbolOff + f.NumSymbols*18
	for _, s := range f.Sections {
		if s.PhysAddr <= entry && entry <= s.PhysAddr+s.Size {
			entrysect = s.Name
			break
		}
	}

	fmt.Printf("\n")
	fmt.Printf("OBJECT FILE:  %v\n", name)
	fmt.Printf("\n")

	var t Textable
	t.SetOutput(os.Stdout)
	t.SetPrespace(4)
	t.SetTitle("Object File Information")
	t.SetSize(13, 2)
	t.SetText(0, 0, "File Name:")
	t.SetText(0, 1, name)
	t.SetText(1, 0, "Time Stamp:")
	t.SetText(1, 1, "%v", time.Unix(int64(f.Timestamp), 0))
	t.SetText(2, 0, "Entry Point:")
	t.SetText(2, 1, "%#08x (%s)", entry, entrysect)
	t.SetText(3, 0, "Number of Sections:")
	t.SetText(3, 1, "%d", f.NumSections)
	t.SetText(4, 0, "File Size:")
	t.SetText(4, 1, "%d", filesz)
	t.SetText(4, 0, "Symbol Table File Offset:")
	t.SetText(4, 1, "%#08x", f.SymbolOff)
	t.SetText(5, 0, "String Table File Offset:")
	t.SetText(5, 1, "%#08x", strtaboff)
	t.SetText(6, 0, "TI-COFF f_flags:")
	t.SetText(6, 1, "%#08x", f.Flags)
	t.SetText(7, 0, "Start of .text section:")
	t.SetText(7, 1, "%#08x", textoff)
	t.SetText(8, 0, "Start of .data section:")
	t.SetText(8, 1, "%#08x", dataoff)
	t.SetText(9, 0, "Size of .text section:")
	t.SetText(9, 1, "%d", textsz)
	t.SetText(10, 0, "Size of .data section:")
	t.SetText(10, 1, "%d", datasz)
	t.SetText(11, 0, "Size of .bss section:")
	t.SetText(11, 1, "%d", bsssz)
	t.Output()
}

func dumpsct(f *coff.File) {
	var t Textable
	t.SetOutput(os.Stdout)
	t.SetPrespace(4)
	t.SetTitle("Section Information")
	t.SetSize(len(f.Sections), 8)
	t.SetHeader(0, "id")
	t.SetHeader(1, "name")
	t.SetHeader(2, "page")
	t.SetHeader(3, "load addr")
	t.SetHeader(4, "run addr")
	t.SetHeader(5, "size")
	t.SetHeader(6, "align")
	t.SetHeader(7, "alloc")
	for i, s := range f.Sections {
		align := (s.Flags >> 8) & 0xf
		switch f.TargetID {
		case coff.MPS430, coff.TMS470:
			align <<= align
		default:
			align = 1 << align
		}

		alloc := 'N'
		if s.Flags&(coff.STYP_REG|coff.STYP_TEXT|coff.STYP_DATA|coff.STYP_BSS) != 0 {
			alloc = 'Y'
		}

		t.SetText(i, 0, "%v", i+1)
		t.SetText(i, 1, "%s", s.Name)
		t.SetText(i, 2, "%d", s.Page)
		t.SetText(i, 3, "%#08x", s.VirtAddr)
		t.SetText(i, 4, "%#08x", s.PhysAddr)
		t.SetText(i, 5, "%#-8x", s.Size)
		t.SetText(i, 6, "%d", align)
		t.SetText(i, 7, "%c", alloc)
	}
	t.Output()
}

func dumpsym(f *coff.File) {
	var t Textable
	t.SetOutput(os.Stdout)
	t.SetPrespace(4)
	t.SetTitle("Symbol Table")
	t.SetSize(0, 7)
	t.SetHeader(0, "id")
	t.SetHeader(1, "name")
	t.SetHeader(2, "value")
	t.SetHeader(3, "kind")
	t.SetHeader(4, "section")
	t.SetHeader(5, "binding")
	t.SetHeader(6, "type")

	for i := 0; i < len(f.Symbols); i++ {
		y := f.Symbols[i]

		sectname := "N/A"
		if int(y.Section-1) < len(f.Sections) {
			sectname = f.Sections[int(y.Section-1)].Name
		}

		typ := "none"
		switch {
		case y.Name == sectname:
			fallthrough
		case strings.HasPrefix(y.Name, "."):
			typ = "section"
		case strings.HasSuffix(y.Name, ".asm"):
			typ = "file"
		case strings.Index(y.Name, "$$") > 0:
			fallthrough
		case strings.HasPrefix(y.Name, "_") && sectname != "N/A":
			typ = "object"
		}

		t.AddText(0, "%d", i)
		t.AddText(1, "%s", y.Name)
		t.AddText(2, "%#08x", y.Value)
		t.AddText(3, "%d", y.Class)
		t.AddText(4, "%s", sectname)
		t.AddText(5, "%s", "_")
		t.AddText(6, "%s", typ)

		i += int(y.Aux)
	}
	t.Output()
}

func dumpstr(f *coff.File) {
	var t Textable
	t.SetOutput(os.Stdout)
	t.SetPrespace(4)
	t.SetTitle("String Table")
	t.SetSize(0, 2)
	t.SetHeader(0, "offset")
	t.SetHeader(1, "string")

	for i := 4; i < len(f.Strings); {
		n := bytes.IndexRune(f.Strings[i:], 0)
		if n < 0 {
			n = len(f.Strings) - i
		}

		t.AddText(0, "%d", i)
		t.AddText(1, "%q", f.Strings[i:i+n])

		i += n + 1
	}
	t.Output()
}

type Textable struct {
	w        io.Writer
	row, col int
	title    string
	header   []string
	table    [][]string
	prespace int
}

func (t *Textable) SetOutput(w io.Writer) {
	t.w = w
}

func (t *Textable) SetTitle(title string) {
	t.title = title
}

func (t *Textable) SetSize(row, col int) {
	t.header = make([]string, col)
	t.table = make([][]string, col)
	for i := range t.table {
		t.table[i] = make([]string, row)
	}
	t.row, t.col = row, col
}

func (t *Textable) SetHeader(col int, header string) {
	t.header[col] = header
}

func (t *Textable) SetPrespace(prespace int) {
	t.prespace = prespace
}

func (t *Textable) SetText(row, col int, format string, args ...interface{}) {
	t.table[col][row] = fmt.Sprintf(format, args...)
}

func (t *Textable) AddText(col int, format string, args ...interface{}) {
	t.table[col] = append(t.table[col], fmt.Sprintf(format, args...))
	t.row = max(t.row, len(t.table[col]))
}

func (t *Textable) Output() error {
	b := bufio.NewWriter(t.w)
	p, hashdr := t.calcspaces()
	fmt.Fprintf(b, " %s\n\n", t.title)

	if hashdr {
		fmt.Fprintf(b, "%*s", t.prespace, " ")
		for i := 0; i < t.col; i++ {
			if len(t.header[i]) > 0 {
				hashdr = true
			}
			fmt.Fprintf(b, "%-*s", p[i], t.header[i])
		}

		fmt.Fprintf(b, "\n")
		fmt.Fprintf(b, "%*s", t.prespace, " ")

		for i := 0; i < t.col; i++ {
			fmt.Fprintf(b, "%-*s", p[i], strings.Repeat("-", len(t.header[i])))
		}
		fmt.Fprintf(b, "\n")
	}

	for i := 0; i < t.row; i++ {
		fmt.Fprintf(b, "%*s", t.prespace, " ")
		for j := 0; j < t.col; j++ {
			fmt.Fprintf(b, "%-*s", p[j], t.table[j][i])
		}
		fmt.Fprintf(b, "\n")
	}

	fmt.Fprintf(b, "\n")
	return b.Flush()
}

func (t *Textable) calcspaces() ([]int, bool) {
	p := make([]int, len(t.header))
	h := false
	for i := 0; i < t.col; i++ {
		p[i] = max(p[i], len(t.header[i])+4)
		if len(t.header[i]) > 0 {
			h = true
		}
	}
	for i := 0; i < t.col; i++ {
		for j := 0; j < t.row; j++ {
			p[i] = max(p[i], len(t.table[i][j])+4)
		}
	}
	return p, h
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
