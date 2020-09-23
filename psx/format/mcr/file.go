package mcr

import (
	"fmt"
	"io"
	"io/ioutil"
)

type File struct {
	Data []byte
}

func Decode(r io.Reader) (*File, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if len(b) < 2 || string(b[:2]) != "MC" {
		return nil, fmt.Errorf("not a valid memory card")
	}

	f := &File{Data: b}
	if len(f.Data) != 0x20000 {
		return nil, fmt.Errorf("invalid file size, expected %d but got %d bytes", len(f.Data), 0x20000)
	}

	return f, nil
}
