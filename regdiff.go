// compare the difference between two register dumps
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"text/template"
)

type Delta struct {
	Addr string
	Val0 string
	Val1 string
}

var flags struct {
	base    int
	width   int
	off     int
	order   binary.ByteOrder
	format  string
	dumpall bool
}

func main() {
	log.SetFlags(0)
	parseflags()

	data0, err0 := os.ReadFile(flag.Arg(0))
	data1, err1 := os.ReadFile(flag.Arg(1))
	check(err0)
	check(err1)

	fmt.Printf("%s | %s\n", flag.Arg(0), flag.Arg(1))
	diff(data0, data1, flags.format, flags.order, flags.base, flags.off, flags.width, flags.dumpall)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: <file1> <file2>")
	flag.PrintDefaults()
	os.Exit(2)
}

func parseflags() {
	var order string
	flag.IntVar(&flags.base, "b", 0, "specify base address")
	flag.IntVar(&flags.width, "w", 4, "specify register width")
	flag.IntVar(&flags.off, "o", 0, "specify offset")
	flag.StringVar(&order, "e", "little", "specify endian")
	flag.StringVar(&flags.format, "f", "", "specify format")
	flag.BoolVar(&flags.dumpall, "d", false, "dump all values")
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 2 {
		usage()
	}

	switch order {
	case "little", "le", "l":
		flags.order = binary.LittleEndian
	case "big", "be", "b":
		flags.order = binary.BigEndian
	default:
		panic("unsupported endian")
	}
}

func diff(data0, data1 []byte, format string, order binary.ByteOrder, base, off, width int, dumpall bool) {
	if len(data0) != len(data1) {
		log.Fatalf("mismatch file size: %d != %d", len(data0), len(data1))
	}

	tmpl := template.New("diff")
	_, err := tmpl.Parse(format)
	check(err)

	size := len(data0)
	for ; off+width-1 < size; off += width {
		val0 := getval(data0[off:], order, width)
		val1 := getval(data1[off:], order, width)
		if val0 != val1 || dumpall {
			if format != "" {
				delta := &Delta{
					Addr: fmt.Sprintf("%#x", base+off),
					Val0: fmt.Sprintf("%#x", val0),
					Val1: fmt.Sprintf("%#x", val1),
				}
				tmpl.Execute(os.Stdout, delta)
				fmt.Println()
			} else {
				fmt.Printf("%#-12x: %#-12x %#-12x\n", base+off, val0, val1)
			}
		}
	}
}

func getval(data []byte, order binary.ByteOrder, width int) uint64 {
	value := uint64(0)
	switch width {
	case 1:
		value = uint64(data[0])
	case 2:
		value = uint64(order.Uint16(data))
	case 4:
		value = uint64(order.Uint32(data))
	case 8:
		value = order.Uint64(data)
	default:
		panic("unsupported width")
	}
	return value
}
