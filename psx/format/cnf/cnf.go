package cnf

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

type File struct {
	Boot  string
	TCB   int
	Event int
	Stack uint32
}

var Defaults = &File{
	Boot:  "cdrom:PSX.EXE;1",
	TCB:   4,
	Event: 5,
	Stack: 0x8001ff00,
}

func Decode(r io.Reader) (*File, error) {
	f := &File{}
	*f = *Defaults
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := s.Text()
		i := strings.Index(line, "=")
		if i <= 0 {
			continue
		}

		key := strings.TrimSpace(strings.ToUpper(line[:i]))
		value := strings.TrimSpace(line[i+1:])
		switch key {
		case "BOOT":
			if value != "" {
				f.Boot = value
			}
		case "TCB":
			f.TCB = int(atoi(value, 0, int64(f.TCB)))
		case "EVENT":
			f.Event = int(atoi(value, 0, int64(f.Event)))
		case "STACK":
			f.Stack = uint32(atoi(value, 16, int64(f.Stack)))
		}
	}

	if err := s.Err(); err != nil {
		return nil, err
	}

	return f, nil
}

func atoi(s string, base int, def int64) int64 {
	n, err := strconv.ParseInt(s, base, 32)
	if err != nil {
		return def
	}
	return n
}
