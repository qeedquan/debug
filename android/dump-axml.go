// based on https://github.com/xgouchet/AXML/
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
)

var (
	status = 0
)

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		ek(dump("<stdin>", os.Stdin))
	} else {
		for _, name := range flag.Args() {
			fd, err := os.Open(name)
			if ek(err) {
				continue
			}
			ek(dump(name, fd))
			fd.Close()
		}
	}
	os.Exit(status)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: dump-axml [options] [file] ...")
	flag.PrintDefaults()
	os.Exit(2)
}

func ek(err error) bool {
	if err != nil {
		fmt.Fprintln(os.Stderr, "dump-axml:", err)
		status = 1
		return true
	}
	return false
}

func dump(name string, r io.Reader) error {
	input, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("%v: %v", name, err)
	}
	d := &dumper{
		input:   input,
		nsp:     make(map[string]string),
		tabstop: 4,
	}
	d.do()
	return nil
}

const (
	WORD_START_DOCUMENT = 0x00080003

	WORD_STRING_TABLE = 0x001C0001
	WORD_RES_TABLE    = 0x00080180

	WORD_START_NS  = 0x00100100
	WORD_END_NS    = 0x00100101
	WORD_START_TAG = 0x00100102
	WORD_END_TAG   = 0x00100103
	WORD_TEXT      = 0x00100104
	WORD_EOS       = 0xFFFFFFFF

	WORD_SIZE = 4
)

const (
	TYPE_ID_REF   = 0x01000008
	TYPE_ATTR_REF = 0x02000008
	TYPE_STRING   = 0x03000008
	TYPE_DIMEN    = 0x05000008
	TYPE_FRACTION = 0x06000008
	TYPE_INT      = 0x10000008
	TYPE_FLOAT    = 0x04000008

	TYPE_FLAGS  = 0x11000008
	TYPE_BOOL   = 0x12000008
	TYPE_COLOR  = 0x1C000008
	TYPE_COLOR2 = 0x1D000008
)

type attribute struct {
	name      string
	prefix    string
	namespace string
	value     string
}

type dumper struct {
	input   []byte
	off     int
	str     []string
	res     []uint32
	nsp     map[string]string
	indent  int
	tabstop int
}

func (d *dumper) do() {
	for d.off < len(d.input) {
		switch tag := d.peek(0); tag {
		case WORD_START_DOCUMENT:
			d.startdoc()
		case WORD_STRING_TABLE:
			d.strtab()
		case WORD_RES_TABLE:
			d.restab()
		case WORD_START_NS:
			d.namespace(true)
		case WORD_END_NS:
			d.namespace(false)
		case WORD_START_TAG:
			d.starttag()
		case WORD_END_TAG:
			d.endtag()
		case WORD_TEXT:
			d.text()
		case WORD_EOS:
			d.enddoc()
		default:
			fmt.Printf("unknown tag %#x\n", tag)
			d.off += WORD_SIZE
		}
	}
}

func (d *dumper) peek(n int) uint32 {
	off := d.off + n*WORD_SIZE
	if off < 0 || off >= len(d.input) || len(d.input)-off < WORD_SIZE {
		return 0
	}
	return binary.LittleEndian.Uint32(d.input[off:])
}

/*
 * A doc starts with the following 4 bytes words:
 * 0th word : 0x00080003
 * 1st word : chunk size
 */
func (d *dumper) startdoc() {
	d.off += 2 * WORD_SIZE
}

/*
 * An end doc starts with the following 4 bytes words:
 * 0th word : 0xFFFFFFFF
 */
func (d *dumper) enddoc() {
	d.off += WORD_SIZE
}

/*
 * the string table starts with the following 4 bytes words:
 * 0th word : 0x1c0001
 * 1st word : chunk size
 * 2nd word : number of string in the string table
 * 3rd word : number of styles in the string table
 * 4th word : flags - sorted/utf8 flag (0)
 * 5th word : Offset to String data
 * 6th word : Offset to style data
 */
func (d *dumper) strtab() {
	chunksz := d.peek(1)
	shoff := uint32(d.off) + d.peek(5)

	d.str = make([]string, d.peek(2))
	for i := range d.str {
		d.str[i] = d.ssnarf(int(shoff + d.peek(i+7)))
	}
	d.off += int(chunksz)
}

/*
 * the resource ids table starts with the following 4bytes words :
 * 0th word : 0x00080180
 * 1st word : chunk size
 */
func (d *dumper) restab() {
	chunksz := d.peek(1)
	d.res = make([]uint32, chunksz/4-2)
	for i := range d.res {
		off := d.off + ((i + 2) * WORD_SIZE)
		d.res[i] = binary.LittleEndian.Uint32(d.input[off:])
	}
	d.off += int(chunksz)
}

/*
 * A namespace tag contains the following 4bytes words :
 * 0th word : 0x00100100 = Start NS / 0x00100101 = end NS
 * 1st word : chunk size
 * 2nd word : line this tag appeared
 * 3rd word : optional xml comment for element (usually 0xFFFFFF)
 * 4th word : index of namespace prefix in StringIndexTable
 * 5th word : index of namespace uri in StringIndexTable
 */
func (d *dumper) namespace(start bool) {
	ns := d.getstr(d.peek(4))
	uri := d.getstr(d.peek(5))
	d.nsp[uri] = ns
	d.off += 6 * WORD_SIZE
}

/*
 * A start tag will start with the following 4 bytes words:
 * 0th word : 0x00100102 = Start_Tag
 * 1st word : chunk size
 * 2nd word : line this tag appeared in the original file
 * 3rd word : optional xml comment for element (usually 0xFFFFFF)
 * 4th word : index of namespace uri in StringIndexTable, or 0xFFFFFFFF for default NS
 * 5th word : index of element name in StringIndexTable
 * 6th word : size of attribute structures to follow
 * 7th word : number of attributes following the start tag
 * 8th word : index of id attribute (0 if none)
 */
func (d *dumper) starttag() {
	ui := d.peek(4)
	ni := d.peek(5)
	ai := d.peek(7)

	uri := ""
	name := d.getstr(ni)
	qname := name
	if ui != 0xffffffff {
		uri = d.getstr(ui)
		if d.nsp[uri] != "" {
			qname = d.nsp[uri] + ":" + name
		}
	}
	d.off += 9 * WORD_SIZE

	attribute := make([]attribute, ai)
	for i := range attribute {
		attribute[i] = d.attribute()
		d.off += 5 * WORD_SIZE
	}

	fmt.Printf("%*.s", d.indent*d.tabstop, " ")
	fmt.Printf("<%s", qname)
	for _, a := range attribute {
		fmt.Printf(" %s=%q", a.name, a.value)
	}
	fmt.Printf(">\n")
	d.indent++
}

/*
 * An attribute will have the following 4bytes words :
 * 0th word : index of namespace uri in StringIndexTable, or 0xFFFFFFFF
 * for default NS
 * 1st word : index of attribute name in StringIndexTable
 * 2nd word : index of attribute value, or 0xFFFFFFFF if value is a
 * typed value
 * 3rd word : value type
 * 4th word : resource id value
 */
func (d *dumper) attribute() attribute {
	nsi := d.peek(0)
	ni := d.peek(1)
	vi := d.peek(2)
	typ := d.peek(3)
	data := d.peek(4)

	var (
		namespace string
		prefix    string
		value     string
	)
	if nsi != 0xffffffff {
		uri := d.getstr(nsi)
		if _, found := d.nsp[uri]; found {
			namespace = uri
			prefix = d.nsp[uri]
		}
	}

	if vi == 0xffffffff {
		value = d.attributeValue(typ, data)
	} else {
		value = d.getstr(vi)
	}

	return attribute{
		name:      d.getstr(ni),
		namespace: namespace,
		prefix:    prefix,
		value:     value,
	}
}

func (d *dumper) attributeValue(typ, data uint32) string {
	dim := []string{
		"px", "dp", "sp", "pt", "in", "mm",
	}
	var s string
	switch typ {
	case TYPE_STRING:
		s = d.getstr(data)
	case TYPE_DIMEN:
		s = fmt.Sprintf("%d %s", data>>8, dim[int(data&0xff)%len(dim)])
	case TYPE_FRACTION:
		s = fmt.Sprintf("%.2f", float64(data)/0x7FFFFFFF)
	case TYPE_FLOAT:
		s = fmt.Sprintf("%f", math.Float32frombits(data))
	case TYPE_INT, TYPE_FLAGS:
		s = fmt.Sprintf("%d", data)
	case TYPE_BOOL:
		if data != 0 {
			s = "true"
		} else {
			s = "false"
		}
	case TYPE_COLOR, TYPE_COLOR2:
		s = fmt.Sprintf("#%08x", data)
	case TYPE_ID_REF:
		s = fmt.Sprintf("@id/%#08x", data)
	case TYPE_ATTR_REF:
		s = fmt.Sprintf("?id/%#08x", data)
	default:
		s = fmt.Sprintf("%#08x/%#08x", typ, data)
	}
	return s
}

/*
 * EndTag contains the following 4bytes words :
 * 0th word : 0x00100103 = End_Tag
 * 1st word : chunk size
 * 2nd word : line this tag appeared in the original file
 * 3rd word : optional xml comment for element (usually 0xFFFFFF)
 * 4th word : index of namespace name in StringIndexTable, or 0xFFFFFFFF for default NS
 * 5th word : index of element name in StringIndexTable
 */
func (d *dumper) endtag() {
	ni := d.peek(5)

	d.indent--
	name := d.getstr(ni)
	fmt.Printf("%*.s</%s>\n", d.indent*d.tabstop, " ", name)

	d.off += 6 * WORD_SIZE
}

/*
 * A text will start with the following 4bytes word :
 * 0th word : 0x00100104 = Text
 * 1st word : chunk size
 * 2nd word : line this element appeared in the original document
 * 3rd word : optional xml comment for element (usually 0xFFFFFF)
 * 4rd word : string index in string table
 * 5rd word : ??? (always 8)
 * 6rd word : ??? (always 0)
 */
func (d *dumper) text() {
	text := d.getstr(uint32(d.off) + d.peek(4))
	fmt.Printf("%*.s%s", d.indent*d.tabstop, " ", text)
	d.off += 7 * WORD_SIZE
}

func (d *dumper) ssnarf(off int) string {
	if off < 0 || off >= len(d.input) || len(d.input)-off < 2 {
		return ""
	}

	var (
		s    string
		n, m int
	)
	if d.input[off] == d.input[off+1] {
		n = int(d.input[off])
		m = 1
	} else {
		n = int(d.input[off]) | int(d.input[off+1])<<8
		m = 2
	}

	for i := 0; i < n; i++ {
		j := off + 2 + i*m
		if j >= len(d.input) {
			break
		}
		s += string(d.input[j])
	}

	return s
}

func (d *dumper) getstr(i uint32) string {
	if i >= uint32(len(d.str)) {
		return ""
	}
	return d.str[i]
}
