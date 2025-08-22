package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/qeedquan/go-media/debug/ti/ais"
)

var (
	dumpdat = flag.Bool("d", false, "dump all data sections")
	remid   = flag.Int("r", -1, "remove command section index")
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("tiaisdump: ")

	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 1 {
		usage()
	}

	m, err := ais.Open(flag.Arg(0))
	ck(err)

	switch {
	case *dumpdat:
		ext := filepath.Ext(flag.Arg(0))
		dir := strings.TrimSuffix(flag.Arg(0), ext)
		dir += "_sections"
		dumpsects(m, dir)
	case *remid >= 0:
		if *remid >= len(m.Cmds) {
			log.Fatal("file does not contain command id %d", *remid)
		}
		fmt.Println("removing command section index", *remid)
		copy(m.Cmds[:*remid], m.Cmds[*remid+1:])
		m.Cmds = m.Cmds[:len(m.Cmds)-1]
		f, err := os.Create(flag.Arg(0))
		ck(err)
		ck(ais.Format(m, f))
		ck(f.Close())
	default:
		dumpcmds(m)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: tiaisdump [options] file")
	flag.PrintDefaults()
	os.Exit(1)
}

func ck(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func dumpcmds(m *ais.Image) {
	for i, c := range m.Cmds {
		fmt.Printf("%4x %v\n", i, ais.Disasm(&c))
	}
}

func dumpsects(m *ais.Image, dir string) {
	os.MkdirAll(dir, 0755)
	for _, c := range m.Cmds {
		switch c.Op {
		case ais.SECTION_LOAD:
			name := filepath.Join(dir, fmt.Sprintf("%x_%x", c.Addr, c.Addr+c.Size-1))
			fmt.Println("writing to", name)
			err := os.WriteFile(name, c.Data, 0644)
			fmt.Fprintln(os.Stderr, "tiaisdump:", err)
		}
	}
}
