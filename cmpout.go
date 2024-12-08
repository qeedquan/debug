package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	nflag = flag.Bool("n", false, "don't output mismatch if text has already been seen")
)

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
	}

	cmp := newComparer()
	for _, arg := range flag.Args() {
		exe := strings.Split(arg, " ")
		cmd := exec.Command(exe[0], exe[1:]...)
		rec := newRecorder(cmp, cmd)
		cmd.Stdout = rec
		cmd.Stderr = rec
		cmp.recs = append(cmp.recs, rec)
	}

	cmp.run()
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: cmpout [options] program ...")
	flag.PrintDefaults()
	os.Exit(2)
}

type comparer struct {
	recs []*recorder
	mis  map[[2]string]bool
}

func newComparer() *comparer {
	return &comparer{
		mis: make(map[[2]string]bool),
	}
}

func (c *comparer) run() {
	for _, r := range c.recs {
		r.cmd.Start()
	}
	for _, r := range c.recs {
		r.cmd.Wait()
	}
}

func (c *comparer) notify(p *recorder) {
	n := len(p.lines) - 1
	l := p.lines[n]
	if len(p.locs[l]) == 1 {
		fmt.Printf("%p %s %d %q\n", p, p.cmd.Path, n+1, l)
	}

	for _, r := range c.recs {
		m := len(r.lines) - 1
		if m < n {
			continue
		}

		rl := r.lines[n]
		if p != r && l != rl {
			k := [2]string{l, rl}
			if _, found := c.mis[k]; !found || !*nflag {
				fmt.Printf("%d: \n", n+1)
				fmt.Printf("  %q\n", l)
				fmt.Printf("  %q\n", rl)
				c.mis[k] = true
			}
		}
	}
}

type recorder struct {
	cmp   *comparer
	cmd   *exec.Cmd
	buf   []byte
	lines []string
	locs  map[string][]int
}

func newRecorder(cmp *comparer, cmd *exec.Cmd) *recorder {
	return &recorder{
		cmp:  cmp,
		cmd:  cmd,
		locs: make(map[string][]int),
	}
}

func (r *recorder) Write(p []byte) (int, error) {
	r.buf = append(r.buf, p...)
	for {
		n := bytes.IndexByte(r.buf, '\n')
		if n < 0 {
			break
		}

		line := string(r.buf[:n])
		r.lines = append(r.lines, line)
		r.buf = r.buf[n+1:]
		r.locs[line] = append(r.locs[line], len(r.lines))

		r.cmp.notify(r)
	}

	return len(p), nil
}
