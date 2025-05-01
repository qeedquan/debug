package main

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/qeedquan/go-binutils/iberty/demangle"
	"golang.org/x/arch/x86/x86asm"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("xeas: ")

	xs := NewXS()
	flag.IntVar(&xs.Mode, "m", xs.Mode, "processor mode")
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
	}

	ck(xs.Open(flag.Arg(0)))
	fn := xs.NewFunc("", flag.Arg(1), flag.Arg(2))
	cg := xs.BuildCallGraph(fn)
	xs.DumpCallGraph(cg)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: xeas [options] file start end")
	flag.PrintDefaults()
	os.Exit(2)
}

func ck(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

const (
	ANONE = iota
	AREG
	AREL
	AMEM
)

type XS struct {
	f      *elf.File
	Mem    []*Mem
	Sym    []*Symbol
	Dynsym []*Symbol
	Mode   int
}

type Mem struct {
	Name  string
	Start uint64
	End   uint64
	Data  []byte
}

type BasicBlock struct {
	Entry uint64
	Exit  uint64
}

type Func struct {
	Name    string
	Start   uint64
	End     uint64
	Dynamic bool
	Inst    []*Inst
	BB      []*BasicBlock
	Callee  []*Func
	Label   []*Symbol
}

type Inst struct {
	x86asm.Inst
	Enc   []byte
	Start uint64
	End   uint64
	Err   error
}

type Symbol struct {
	elf.Symbol
	Dynamic bool
	Label   bool
}

func NewXS() *XS {
	return &XS{
		Mode: 64,
	}
}

func (xs *XS) Open(name string) error {
	var err error

	xs.f, err = elf.Open(name)
	if err != nil {
		return err
	}

	sym, _ := xs.f.Symbols()
	sort.SliceStable(sym, func(i, j int) bool {
		return sym[i].Value < sym[j].Value
	})
	xs.Sym = xs.Sym[:0]
	for i := range sym {
		xs.Sym = append(xs.Sym, &Symbol{
			Symbol:  sym[i],
			Dynamic: false,
		})
	}

	for _, p := range xs.f.Progs {
		if p.Type != elf.PT_LOAD {
			continue
		}
		m := xs.Mmap(name, p.Vaddr, p.Memsz)
		Data, err := io.ReadAll(p.Open())
		if err != nil {
			return fmt.Errorf("failed to load segment: %v", err)
		}
		copy(m.Data, Data)
	}
	if len(xs.Mem) == 0 {
		return fmt.Errorf("executable has no loadable segment")
	}

	dynsym, _ := xs.f.DynamicSymbols()
	for _, p := range xs.f.Sections {
		if p.Type != elf.SHT_RELA {
			continue
		}
		q := xs.f.Section(strings.TrimPrefix(p.Name, ".rela"))
		if q == nil {
			continue
		}

		data, err := p.Data()
		if err != nil {
			return fmt.Errorf("failed to get relative section: %v", err)
		}

		rd := bytes.NewReader(data)
		for {
			var (
				rel uint64
				idx uint64
			)
			switch xs.f.Class {
			case elf.ELFCLASS64:
				var v elf.Rela64
				err = binary.Read(rd, xs.f.ByteOrder, &v)
				rel = v.Info & 0xffffffff
				idx = v.Info >> 32
			case elf.ELFCLASS32:
				var v elf.Rela32
				err = binary.Read(rd, xs.f.ByteOrder, &v)
				rel = uint64(v.Info & 0xffff)
				idx = uint64(v.Info >> 16)
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			if idx >= uint64(len(dynsym)) {
				continue
			}

			switch {
			case rel == uint64(elf.R_X86_64_JMP_SLOT),
				rel == uint64(elf.R_386_JMP_SLOT):
				ds := &dynsym[idx-1]
				ds.Value = q.Addr + q.Entsize*idx
				ds.Size = q.Entsize
			}
		}
	}
	for i := range dynsym {
		xs.Dynsym = append(xs.Dynsym, &Symbol{
			Symbol:  dynsym[i],
			Dynamic: true,
		})
	}

	return nil
}

func (xs *XS) Mmap(name string, addr, size uint64) *Mem {
	m := &Mem{
		Name:  name,
		Start: addr,
		End:   addr + size,
		Data:  make([]byte, size),
	}
	for _, mp := range xs.Mem {
		if mp.Start <= m.Start && m.Start <= mp.End ||
			mp.Start <= m.End && m.End <= mp.End {
			log.Fatalf("mmap: overlapping Memory: %q %x-%x %x-%x\n",
				name, m.Start, m.End, mp.Start, mp.End)
		}
	}
	xs.Mem = append(xs.Mem, m)
	sort.Slice(xs.Mem, func(i, j int) bool {
		return xs.Mem[i].Start < xs.Mem[j].Start
	})
	return m
}

func (xs *XS) symAt(addr uint64) (y *Symbol, n int) {
	n = -1
	for i, s := range xs.Sym {
		if s.Value <= addr {
			y = xs.Sym[i]
			n = i
		}
	}
	return
}

func (xs *XS) NewFunc(name, sp, ep string) *Func {
	var start, end uint64
	start, sdx := xs.ftoi(sp, xs.f.Entry)
	if sdx >= 0 {
		if xs.Sym[sdx].Size != 0 {
			end = start + xs.Sym[sdx].Size
		} else if sdx+1 < len(xs.Sym) {
			end = xs.Sym[sdx+1].Value
		}
	} else {
		if _, i := xs.symAt(start); i+1 < len(xs.Sym) {
			end = xs.Sym[i+1].Value
		} else {
			end = xs.Mem[len(xs.Mem)-1].End
		}
	}

	if ep != "" {
		end, _ = xs.ftoi(ep, end)
	}

	if name == "" {
		if isAlpha(sp) {
			name = sp
		} else {
			if s, _ := xs.symAt(start); s != nil {
				name = s.Name
			} else {
				name = fmt.Sprintf("func_%x", start)
			}
		}
	}

	dynamic := false
	sym, _ := xs.lookupAddr(start)
	if sym != nil {
		if sym.Dynamic {
			dynamic = true
		}
	}

	inst := xs.fetchv(start, end)
	bb := xs.genBB(inst)
	callee := xs.genCalls(start, inst)
	label := xs.genLabel(inst)
	return &Func{
		Name:    name,
		Start:   start,
		End:     end,
		Inst:    inst,
		BB:      bb,
		Callee:  callee,
		Label:   label,
		Dynamic: dynamic,
	}
}

func (xs *XS) ftoi(str string, def uint64) (uint64, int) {
	if str == "" {
		return def, -1
	}

	n, err := strconv.ParseUint(str, 0, 64)
	if err == nil {
		return n, -1
	}

	for i, s := range xs.Sym {
		if s.Name == str {
			return s.Value, i
		}
	}

	log.Fatalf("unable to find symbol %q", str)
	panic("unreachable")
}

func (xs *XS) Data(addr, size uint64) []byte {
	b := make([]byte, size)
	n := uint64(0)
	for _, m := range xs.Mem {
		if m.Start <= addr && addr <= m.End {
			i := addr - m.Start
			j := i + size - n
			if k := uint64(len(m.Data)); j > k {
				j = k
			}
			n += uint64(copy(b, m.Data[i:j]))
		}
	}

	if n == 0 {
		return nil
	}
	return b[:n]
}

func (xs *XS) fetch(ip uint64) (*Inst, error) {
	code := xs.Data(ip, 16)
	if code == nil {
		return nil, fmt.Errorf("failed to fetch instruction at unmapped memory %#x", ip)
	}
	if len(code) == 0 {
		return nil, io.EOF
	}

	i, err := x86asm.Decode(code, xs.Mode)
	return &Inst{
		Inst:  i,
		Start: ip,
		End:   ip + uint64(i.Len),
		Enc:   code[:i.Len],
	}, err
}

func (xs *XS) fetchv(start, end uint64) []*Inst {
	insts := make([]*Inst, 0, 256)
	ip := start
	for ip < end {
		inst, err := xs.fetch(ip)
		if err != nil {
			insts = append(insts, &Inst{
				Err: err,
			})
			break
		}
		insts = append(insts, inst)
		ip += uint64(inst.Len)
	}

	return insts
}

func (xs *XS) genBB(inst []*Inst) []*BasicBlock {
	var bb []*BasicBlock

	i := 0
	for j, inst := range inst {
		if isBranch(inst) {
			bb = append(bb, &BasicBlock{uint64(i), uint64(j)})
			i = j + 1
		}
	}
	return bb
}

func (xs *XS) genLabel(inst []*Inst) []*Symbol {
	if len(inst) == 0 {
		return nil
	}

	var label []*Symbol
	start := inst[0].Start
	end := inst[len(inst)-1].End
loop:
	for _, inst := range inst {
		if !isBranch(inst) && !isRel(inst) {
			continue
		}

		addr := getRel(inst)
		if start <= addr && addr < end {
			for i := range label {
				if label[i].Value == addr {
					continue loop
				}
			}
			label = append(label, &Symbol{
				Symbol: elf.Symbol{
					Value: addr,
				},
			})
		}
	}
	sort.SliceStable(label, func(i, j int) bool {
		return label[i].Value < label[j].Value
	})

	return label
}

func (xs *XS) genCalls(start uint64, inst []*Inst) []*Func {
	var fn []*Func
	addr := start
	for _, inst := range inst {
		if inst.Op != x86asm.CALL {
			addr += uint64(inst.Len)
			continue
		}

		switch a := inst.Args[0].(type) {
		case x86asm.Rel:
			ip := addr + uint64(inst.Len) + uint64(a)
			sym, _ := xs.lookupAddr(ip)
			if sym == nil {
				sym = &Symbol{
					Symbol: elf.Symbol{
						Name:  fmt.Sprintf("func%x", ip),
						Value: ip,
						Size:  32,
					},
				}
			}
			fn = append(fn, &Func{
				Name:    sym.Name,
				Start:   sym.Value,
				End:     sym.Value + sym.Size,
				Dynamic: sym.Dynamic,
			})
		}

		addr += uint64(inst.Len)
	}
	return fn
}

func (xs *XS) lookupAddr(addr uint64) (*Symbol, int) {
	for _, p := range [][]*Symbol{xs.Sym, xs.Dynsym} {
		for i, s := range p {
			if s.Value <= addr && addr < s.Value+s.Size {
				return s, i
			}
		}
	}
	return nil, -1
}

func (xs *XS) BuildCallGraph(root *Func) map[string]*Func {
	cg := make(map[string]*Func)
	fl := []*Func{root}
	for ; len(fl) > 0; fl = fl[1:] {
		fn := fl[0]
		if _, found := cg[fn.Name]; found {
			continue
		}

		cg[fn.Name] = fn
		for _, c := range fn.Callee {
			fl = append(fl, xs.NewFunc(c.Name, fmt.Sprint(c.Start), fmt.Sprint(c.End)))
		}
	}
	return cg
}

func (xs *XS) DumpCallGraph(cg map[string]*Func) {
	var fn []*Func
	for _, f := range cg {
		fn = append(fn, f)
	}
	sort.SliceStable(fn, func(i, j int) bool {
		if fn[i].Dynamic && !fn[j].Dynamic {
			return true
		}
		if fn[j].Dynamic && !fn[i].Dynamic {
			return false
		}
		return fn[i].Name < fn[j].Name
	})

	w := os.Stdout
	for _, f := range fn {
		if !f.Dynamic {
			fmt.Fprintf(w, ".globl %s\n", f.Name)
		}
	}
	fmt.Fprintf(w, "\n")
	for _, f := range fn {
		addr := f.Start

		cname := demangle.Cplus(f.Name, demangle.PARAMS|demangle.TYPES|demangle.VERBOSE)
		if cname == "" {
			cname = f.Name
		}
		fmt.Fprintf(w, "# %s %#x:%#x\n", cname, f.Start, f.End)
		if f.Dynamic {
			fmt.Fprintf(w, "# dynamic function\n")
		}
		fmt.Fprintf(w, "%s:\n", f.Name)
		if f.Dynamic {
			fmt.Fprintf(w, "\t%-64s # %#x\n\n", "retq", addr)
			continue
		}

		l := 0
		for _, h := range f.Inst {
			if l < len(f.Label) && f.Label[l].Value <= h.Start {
				fmt.Fprintf(w, "label_%x:\n", f.Label[l].Value)
				l++
			}
			fmt.Fprintf(w, "\t%s\n", xs.syntax(h))
			addr += uint64(h.Len)
		}
		fmt.Fprintf(w, "\n")
	}
}

func (xs *XS) syntax(inst *Inst) string {
	var str string
	if inst.Err != nil {
		str = fmt.Sprintf("# %v", inst.Err.Error())
	} else {
		switch inst.Op {
		case x86asm.CALL:
			switch {
			case isRel(inst):
				addr := getRel(inst)
				sym, _ := xs.lookupAddr(addr)
				if sym != nil {
					str = fmt.Sprintf("call %s", sym.Name)
				} else {
					str = fmt.Sprintf("call func_%x", addr)
				}
			default:
				str = x86asm.GNUSyntax(inst.Inst, 0, nil)
			}
		case x86asm.JMP, x86asm.JNE, x86asm.JE, x86asm.JA,
			x86asm.JB, x86asm.JBE, x86asm.JAE, x86asm.JLE,
			x86asm.JGE, x86asm.JG, x86asm.JS, x86asm.JNS:
			switch {
			case isRel(inst):
				rel := getRel(inst)
				op := strings.ToLower(inst.Op.String())
				str = fmt.Sprintf("%s label_%x", op, rel)
			}
		default:
			str = x86asm.GNUSyntax(inst.Inst, 0, nil)
		}
	}
	return fmt.Sprintf("%-64s # %#x % x", str, inst.Start, inst.Enc)
}

func isArg(inst *Inst, typ ...int) bool {
	var v int
	for i, a := range inst.Args {
		if i >= len(typ) {
			break
		}
		switch a.(type) {
		case x86asm.Reg:
			v = AREG
		case x86asm.Rel:
			v = AREL
		case x86asm.Mem:
			v = AMEM
		default:
			panic(fmt.Errorf("unhandled type %T", a))
		}

		if v != typ[i] {
			return false
		}
	}
	return true
}

func isBranch(inst *Inst) bool {
	switch inst.Op {
	case x86asm.JMP, x86asm.JNE, x86asm.JE, x86asm.JA,
		x86asm.JB, x86asm.JBE, x86asm.JAE, x86asm.JLE,
		x86asm.JGE, x86asm.JG, x86asm.JS, x86asm.JNS:
		return true
	}
	return false
}

func isRel(inst *Inst) bool {
	return (isBranch(inst) || inst.Op == x86asm.CALL) && isArg(inst, AREL)
}

func isAlpha(str string) bool {
	if strings.TrimSpace(str) == "" {
		return false
	}

	_, err := strconv.ParseInt(str, 0, 64)
	if err != nil {
		return true
	}
	return false
}

func getRel(inst *Inst) uint64 {
	if !isRel(inst) {
		return 0
	}
	return uint64(int64(inst.End) + int64(inst.Args[0].(x86asm.Rel)))
}
