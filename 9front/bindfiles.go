package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
)

type File struct {
	os.DirEntry
	Path  string
	Xpath string
	Uniq  bool
}

var (
	xsrc = flag.String("s", "", "use path relative to source when printing")
	xdst = flag.String("d", "/", "use path relative to destination when printing")
)

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 2 {
		usage()
	}

	bind(flag.Arg(0), flag.Arg(1), *xsrc, *xdst)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: bindfiles [options] from to")
	flag.PrintDefaults()
	os.Exit(2)
}

func ck(err error) {
	if err != nil {
		log.Fatal("bindfiles:", err)
	}
}

func bind(from, to, xfrom, xto string) {
	if xfrom == "" {
		xfrom = from
	}
	if xto == "" {
		xto = to
	}

	var (
		df, sf   [][2]string
		worklist [][2]File
	)
	for {
		src := readdir(from, xfrom)
		dst := readdir(to, xto)
		files, uniqs := filter(src, dst)
		if uniqs {
			df = append(df, [2]string{xfrom, xto})
		}

		for _, f := range files {
			if !f[0].IsDir() && !f[0].Uniq {
				sf = append(sf, [2]string{f[0].Xpath, f[1].Xpath})
			}

			if f[0].IsDir() && !f[0].Uniq {
				worklist = append(worklist, f)
			}
		}

		if len(worklist) == 0 {
			break
		}

		from = worklist[0][0].Path
		to = worklist[0][1].Path
		xfrom = worklist[0][0].Xpath
		xto = worklist[0][1].Xpath
		worklist = worklist[1:]
	}

	sort.Slice(df, func(i, j int) bool {
		return df[i][0] < df[j][0]
	})
	sort.Slice(sf, func(i, j int) bool {
		return sf[i][0] < sf[i][0]
	})

	for _, f := range df {
		fmt.Printf("bind -ac %s %s\n", f[0], f[1])
	}
	for _, f := range sf {
		fmt.Printf("bind %s %s\n", f[0], f[1])
	}
}

func readdir(dir, xdir string) []File {
	fis, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("# %v", err)
		return nil
	}

	var files []File
	for _, fi := range fis {
		files = append(files, File{
			DirEntry: fi,
			Path:     path.Join(dir, fi.Name()),
			Xpath:    path.Join(xdir, fi.Name()),
		})
	}
	return files
}

func filter(a, b []File) (files [][2]File, uniqs bool) {
loop:
	for _, x := range a {
		for _, y := range b {
			if x.Name() == y.Name() {
				files = append(files, [2]File{x, y})
				continue loop
			}
		}

		x.Uniq = true
		files = append(files, [2]File{x, File{}})
		uniqs = true
	}
	return
}
