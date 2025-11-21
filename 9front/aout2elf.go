package main

import (
	"bufio"
	"bytes"
	"debug/elf"
	"debug/plan9obj"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
)

var (
	textaddr = flag.Int64("T", -1, "specify start address of text section")
	dataaddr = flag.Int64("D", -1, "specify start address of data section")
	output   = flag.String("o", "", "output to file")
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
	}

	if *output == "" {
		base := filepath.Base(flag.Arg(0))
		ext := filepath.Ext(base)
		*output = base[:len(base)-len(ext)] + ".elf"
		if strings.ToLower(ext) == ".elf" {
			*output += ".elf"
		}
	}

	ck(convert(flag.Arg(0), *output))
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: aout2elf [options] file")
	flag.PrintDefaults()
	os.Exit(2)
}

func ck(err error) {
	if err != nil {
		log.Fatal("aout2elf: ", err)
	}
}

func align(x, a uint64) uint64 {
	return (x + (a - 1)) &^ (a - 1)
}

func convert(input, output string) error {
	r, err := plan9obj.Open(input)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := os.Create(output)
	if err != nil {
		return err
	}

	c := conv{r: r, w: bufio.NewWriter(w)}
	err = c.do()
	xerr := w.Close()
	if err == nil {
		err = xerr
	}
	return err
}

type conv struct {
	r      *plan9obj.File
	w      *bufio.Writer
	order  binary.ByteOrder
	err    error
	hdrsz  uint64
	hdrpad uint64
	text   *plan9obj.Section
	textld uint64
	data   *plan9obj.Section
	datald uint64
	syms   []plan9obj.Sym
	symb   *bytes.Buffer
	symsz  uint64
	symstr uint64
}

func (c *conv) do() error {
	c.writehdr()
	c.writephdr()
	c.writesect()
	c.writeshdr()

	err := c.w.Flush()
	if err == nil {
		err = c.err
	}
	return err
}

func (c *conv) writehdr() {
	c.order = binary.LittleEndian

	c.text = c.r.Section("text")
	c.data = c.r.Section("data")
	if c.text == nil || c.data == nil {
		c.err = fmt.Errorf("binary missing text/data section")
		return
	}
	c.writesyms()

	switch c.r.Magic {
	case plan9obj.Magic386, plan9obj.MagicARM:
		c.hdrsz = align(0x34+0x20*3, 16)
		c.hdrpad = c.hdrsz - (0x34 + 0x20*3)

		machine := uint16(0x3)
		if c.r.Magic == plan9obj.MagicARM {
			machine = 0x28
		}

		c.write(elf.Header32{
			Ident:     [elf.EI_NIDENT]byte{0x7f, 'E', 'L', 'F', 0x1, 0x1, 0x1},
			Type:      2,
			Machine:   machine,
			Version:   1,
			Entry:     uint32(c.r.Entry),
			Phoff:     0x34,
			Ehsize:    0x34,
			Phentsize: 0x20,
			Phnum:     3,
			Shoff:     uint32(c.hdrsz + uint64(c.text.Size+c.data.Size) + c.symsz),
			Shentsize: 0x28,
			Shnum:     5,
			Shstrndx:  4,
		})

	case plan9obj.MagicAMD64:
		c.hdrsz = align(0x40+0x38*3, 16)
		c.hdrpad = c.hdrsz - (0x40 + 0x38*3)
		c.write(elf.Header64{
			Ident:     [elf.EI_NIDENT]byte{0x7f, 'E', 'L', 'F', 0x2, 0x1, 0x1},
			Type:      2,
			Machine:   0x3e,
			Version:   1,
			Entry:     c.r.Entry,
			Phoff:     0x40,
			Ehsize:    0x40,
			Phentsize: 0x38,
			Phnum:     3,
			Shoff:     c.hdrsz + uint64(c.text.Size+c.data.Size) + c.symsz,
			Shentsize: 0x40,
			Shnum:     5,
			Shstrndx:  4,
		})
	default:
		c.err = errors.New("unsupported a.out format")
	}
}

func (c *conv) writesyms() {
	c.syms, _ = c.r.Symbols()
	c.symstr = 29

	c.textld = math.MaxUint64
	c.datald = math.MaxUint64
	for _, s := range c.syms {
		switch s.Type {
		case 't', 'T', 'l', 'L':
			if c.textld > s.Value {
				c.textld = s.Value
			}
		case 'd', 'b', 'D', 'B':
			if c.datald > s.Value {
				c.datald = s.Value
			}
		}
	}
	if c.textld == math.MaxUint64 {
		c.textld = c.r.LoadAddress
	}
	if c.datald == math.MaxUint64 {
		c.datald = c.textld + uint64(c.text.Size)
	}

	if *textaddr >= 0 {
		c.textld = uint64(*textaddr)
	}
	if *dataaddr >= 0 {
		c.datald = uint64(*dataaddr)
	}

	c.symb = new(bytes.Buffer)
	c.writesym(c.symb, 0, elf.Symbol{})
	c.writesym(c.symb, 1, elf.Symbol{
		Info:  byte(elf.STT_SECTION),
		Value: c.textld,
	})
	c.writesym(c.symb, 7, elf.Symbol{
		Info:  byte(elf.STT_SECTION),
		Value: c.datald,
	})
loop:
	for _, s := range c.syms {
		var (
			info byte
			sect int
		)

		value := s.Value
		switch s.Type {
		case 'd', 'b':
			info = byte(elf.STT_OBJECT) | byte(elf.STB_LOCAL)<<4
			sect = 2

		case 'D', 'B':
			info = byte(elf.STT_OBJECT) | byte(elf.STB_GLOBAL)<<4
			sect = 2

		case 't', 'l':
			info = byte(elf.STT_FUNC) | byte(elf.STB_LOCAL)<<4
			sect = 1

		case 'T', 'L':
			info = byte(elf.STT_FUNC) | byte(elf.STB_GLOBAL)<<4
			sect = 1

		case 'z', 'Z':
			info = byte(elf.STT_FILE) | byte(elf.STB_GLOBAL)<<4

		default:
			c.symstr += uint64(len(s.Name)) + 1
			continue loop
		}

		c.writesym(c.symb, uint32(c.symstr), elf.Symbol{
			Info:    info,
			Section: elf.SectionIndex(sect),
			Value:   value,
		})
		c.symstr += uint64(len(s.Name)) + 1
	}
	c.symsz = uint64(c.symb.Len())
}

func (c *conv) writephdr() {
	off := c.hdrsz
	addr := c.textld

	// text
	c.writexphdr(elf.ProgHeader{
		Type:   elf.PT_LOAD,
		Filesz: uint64(c.text.Size),
		Memsz:  uint64(c.text.Size),
		Align:  4,
		Vaddr:  addr,
		Paddr:  addr,
		Flags:  elf.PF_X | elf.PF_R,
		Off:    off,
	})
	off += uint64(c.text.Size)
	addr = c.datald

	// data
	c.writexphdr(elf.ProgHeader{
		Type:   elf.PT_LOAD,
		Filesz: uint64(c.data.Size),
		Memsz:  uint64(c.data.Size) + uint64(c.r.Bss),
		Vaddr:  addr,
		Paddr:  addr,
		Align:  4,
		Flags:  elf.PF_R | elf.PF_W,
		Off:    off,
	})
	off += uint64(c.data.Size)

	// symbol table
	c.writexphdr(elf.ProgHeader{
		Type:   elf.PT_NULL,
		Filesz: c.symsz,
		Memsz:  c.symsz,
		Align:  4,
		Flags:  elf.PF_R,
		Off:    off,
	})

	for i := uint64(0); i < c.hdrpad; i++ {
		c.write(byte(0))
	}
}

func (c *conv) writexphdr(p elf.ProgHeader) {
	switch c.r.Magic {
	case plan9obj.Magic386, plan9obj.MagicARM:
		c.write(elf.Prog32{
			Type:   uint32(p.Type),
			Flags:  uint32(p.Flags),
			Off:    uint32(p.Off),
			Vaddr:  uint32(p.Vaddr),
			Paddr:  uint32(p.Paddr),
			Filesz: uint32(p.Filesz),
			Memsz:  uint32(p.Memsz),
			Align:  uint32(p.Align),
		})
	case plan9obj.MagicAMD64:
		c.write(elf.Prog64{
			Type:   uint32(p.Type),
			Flags:  uint32(p.Flags),
			Off:    p.Off,
			Vaddr:  p.Vaddr,
			Paddr:  p.Paddr,
			Filesz: p.Filesz,
			Memsz:  p.Memsz,
			Align:  p.Align,
		})
	}
}

func (c *conv) writesect() {
	io.Copy(c.w, c.text.Open())
	io.Copy(c.w, c.data.Open())
	io.Copy(c.w, c.symb)
}

func (c *conv) writeshdr() {
	addr := c.textld
	off := c.hdrsz

	// null
	c.writexshdr(0, elf.SectionHeader{})

	// text
	c.writexshdr(1, elf.SectionHeader{
		Type:      elf.SHT_PROGBITS,
		Addr:      addr,
		Offset:    off,
		Size:      uint64(c.text.Size),
		Flags:     elf.SHF_ALLOC | elf.SHF_EXECINSTR,
		Addralign: 4,
	})
	off += uint64(c.text.Size)
	addr = c.datald

	// data
	c.writexshdr(7, elf.SectionHeader{
		Type:      elf.SHT_PROGBITS,
		Addr:      addr,
		Offset:    off,
		Size:      uint64(c.data.Size),
		Flags:     elf.SHF_ALLOC | elf.SHF_WRITE,
		Addralign: 4,
	})
	off += uint64(c.data.Size)

	// symtab
	var entsize uint64
	switch c.r.Magic {
	case plan9obj.Magic386, plan9obj.MagicARM:
		entsize = 0x10
	case plan9obj.MagicAMD64:
		entsize = 0x18
	}
	c.writexshdr(13, elf.SectionHeader{
		Type:      elf.SHT_SYMTAB,
		Offset:    off,
		Size:      c.symsz,
		Addralign: 1,
		Link:      4,
		Entsize:   entsize,
	})
	off += c.symsz
	switch c.r.Magic {
	case plan9obj.MagicARM, plan9obj.Magic386:
		off += 0x28 * 5
	case plan9obj.MagicAMD64:
		off += 0x40 * 5
	}

	// strtab
	c.writexshdr(21, elf.SectionHeader{
		Type:      elf.SHT_STRTAB,
		Offset:    off,
		Size:      c.symstr,
		Flags:     elf.SHF_STRINGS,
		Addralign: 1,
	})

	c.writestrz("")
	c.writestrz(".text")
	c.writestrz(".data")
	c.writestrz(".symtab")
	c.writestrz(".strtab")
	for _, s := range c.syms {
		c.writestrz(s.Name)
	}
}

func (c *conv) writexshdr(name uint32, p elf.SectionHeader) {
	switch c.r.Magic {
	case plan9obj.Magic386, plan9obj.MagicARM:
		c.write(elf.Section32{
			Name:      name,
			Type:      uint32(p.Type),
			Flags:     uint32(p.Flags),
			Addr:      uint32(p.Addr),
			Off:       uint32(p.Offset),
			Size:      uint32(p.Size),
			Link:      p.Link,
			Info:      p.Info,
			Addralign: uint32(p.Addralign),
			Entsize:   uint32(p.Entsize),
		})
	case plan9obj.MagicAMD64:
		c.write(elf.Section64{
			Name:      name,
			Type:      uint32(p.Type),
			Flags:     uint64(p.Flags),
			Addr:      p.Addr,
			Off:       p.Offset,
			Size:      p.Size,
			Link:      p.Link,
			Info:      p.Info,
			Addralign: p.Addralign,
			Entsize:   p.Entsize,
		})
	}
}

func (c *conv) writesym(b *bytes.Buffer, name uint32, p elf.Symbol) {
	switch c.r.Magic {
	case plan9obj.MagicARM, plan9obj.Magic386:
		binary.Write(b, c.order, elf.Sym32{
			Name:  name,
			Info:  p.Info,
			Other: p.Other,
			Shndx: uint16(p.Section),
			Value: uint32(p.Value),
			Size:  uint32(p.Size),
		})
	case plan9obj.MagicAMD64:
		binary.Write(b, c.order, elf.Sym64{
			Name:  name,
			Info:  p.Info,
			Other: p.Other,
			Shndx: uint16(p.Section),
			Value: p.Value,
			Size:  p.Size,
		})
	}
}

func (c *conv) write(v interface{}) {
	if c.err != nil {
		return
	}
	binary.Write(c.w, c.order, v)
}

func (c *conv) writestrz(str string) {
	for i := 0; i < len(str); i++ {
		c.write(byte(str[i]))
	}
	c.write(byte(0))
}
