package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	parseflags()

	w := new(bytes.Buffer)
	for _, arg := range flag.Args() {
		v, err := strconv.ParseInt(arg, 0, 64)
		ck(err)

		gen(w, v)
	}

	b := w.Bytes()
	s := fmt.Sprintf("asm-%d.s", rand.Int())
	t := fmt.Sprintf("asm-%d.o", rand.Int())
	os.WriteFile(s, b, 0644)
	runcmd("as", "-o", t, s)
	runcmd("objdump", "-D", t)
	os.Remove(s)
	os.Remove(t)
}

func parseflags() {
	flag.Usage = usage
	flag.Parse()
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: <offset> ...")
}

func ck(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func gen(w io.Writer, v int64) {
	regs := []string{
		"rax", "rbx", "rcx", "rdi", "rsi", "rbp",
		"eax", "ebx", "ecx", "edi", "esi", "ebp",
	}
	for _, r := range regs {
		fmt.Fprintf(w, "mov $%#x, %%%s\n", v, r)
	}
}

func runcmd(c string, a ...string) {
	cmd := exec.Command(c, a...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Run()
}
