package exe

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	"format/coff"
)

var (
	errMagic = errors.New("mismatched magic")
)

type Section struct {
	Name string
	Addr uint32
	Data []byte
}

type PSX struct {
	Sig       [8]byte
	Text      uint32
	Data      uint32
	PC0       uint32
	GP0       uint32
	TextAddr  uint32
	TextSize  uint32
	DataAddr  uint32
	DataSize  uint32
	BSSAddr   uint32
	BSSSize   uint32
	StackAddr uint32
	StackSize uint32
	SP        uint32
	FP        uint32
	GP        uint32
	RA        uint32
	S0        uint32
	Sections  []Section
}

type SCE struct {
	Sig       [8]byte
	PC0       uint32
	GP0       uint32
	TextAddr  uint32
	TextSize  uint32
	StackAddr uint32
	StackSize uint32
	SP        uint32
	FP        uint32
	GP        uint32
	RA        uint32
	S0        uint32
	Sections  []Section
}

type CPE struct {
	PC       uint32
	Sections []Section
}

type File struct {
	GPR      [32]uint32
	GP0      uint32
	PC0      uint32
	Sections []Section
}

func Decode(r io.Reader) (*File, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	psx, err := DecodePSX(bytes.NewReader(b))
	switch {
	case err != nil && err != errMagic:
		return nil, err
	case err == nil:
		return &File{
			GPR: [32]uint32{
				16: psx.S0,
				28: psx.GP,
				29: psx.SP,
				30: psx.FP,
				31: psx.RA,
			},
			GP0:      psx.GP0,
			PC0:      psx.PC0,
			Sections: psx.Sections,
		}, nil
	}

	sce, err := DecodeSCE(bytes.NewReader(b))
	switch {
	case err != nil && err != errMagic:
		return nil, err
	case err == nil:
		return &File{
			GPR: [32]uint32{
				16: sce.S0,
				28: sce.GP,
				29: sce.SP,
				30: sce.FP,
				31: sce.RA,
			},
			GP0:      sce.GP0,
			PC0:      sce.PC0,
			Sections: sce.Sections,
		}, nil
	}

	cpe, err := DecodeCPE(bytes.NewReader(b))
	switch {
	case err != nil && err != errMagic:
		return nil, err
	case err == nil:
		return &File{
			PC0:      cpe.PC,
			Sections: cpe.Sections,
		}, nil
	}

	if len(b) > 2 && b[0] == 0x62 && b[1] == 0x01 {
		c, err := coff.Decode(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}

		f := &File{
			PC0: c.Entry,
		}
		for _, s := range c.Sections {
			f.Sections = append(f.Sections, Section{
				Name: strings.TrimRight(string(s.Name[:]), "\x00"),
				Addr: s.Paddr,
				Data: s.Data,
			})
		}

		return f, nil
	}

	return nil, errors.New("unknown executable format")
}

func DecodePSX(r io.Reader) (*PSX, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	p := &PSX{}
	err = decodeStruct(b, p)
	if err != nil {
		return nil, err
	}

	if string(p.Sig[:]) != "PS-X EXE" {
		return nil, errMagic
	}

	p.Sections, err = decodeTextSection(b, p.TextAddr, p.TextSize)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func DecodeSCE(r io.Reader) (*SCE, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	p := &SCE{}
	err = decodeStruct(b, p)
	if err != nil {
		return nil, err
	}

	if string(p.Sig[:]) != "SCE EXE\x00" {
		return nil, errMagic
	}

	p.Sections, err = decodeTextSection(b, p.TextAddr, p.TextSize)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func DecodeCPE(r io.Reader) (*CPE, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if len(b) < 6 || string(b[:3]) != "CPE" {
		return nil, errMagic
	}

	p := &CPE{}
	br := bytes.NewReader(b[6:])
	for {
		op, err := br.ReadByte()
		if err != nil {
			return nil, err
		}
		if op == 0 {
			break
		}

		switch op {
		case 1: // section loading
			var addr, size uint32
			err = binary.Read(br, binary.LittleEndian, &addr)
			if err != nil {
				return nil, err
			}

			err = binary.Read(br, binary.LittleEndian, &size)
			if err != nil {
				return nil, err
			}

			data := make([]byte, size)
			_, err = io.ReadAtLeast(br, data, len(data))
			if err != nil {
				return nil, err
			}

			p.Sections = append(p.Sections, Section{
				Addr: addr,
				Data: data,
			})

		case 3: // register loading (PC only?)
			_, err = br.Seek(2, os.SEEK_CUR)
			if err != nil {
				return nil, err
			}
			binary.Read(br, binary.LittleEndian, &p.PC)

		default:
			return nil, fmt.Errorf("invalid opcode %#x\n", op)
		}
	}

	return p, nil
}

func decodeStruct(b []byte, x interface{}) error {
	r := bytes.NewReader(b)
	rv := reflect.ValueOf(x).Elem()
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)
		switch f.Kind() {
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Array:
			v := f.Addr().Interface()
			err := binary.Read(r, binary.LittleEndian, v)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func decodeTextSection(b []byte, addr, size uint32) ([]Section, error) {
	if len(b) <= 2048 {
		return nil, io.ErrShortBuffer
	}

	if len(b)%2048 != 0 {
		return nil, fmt.Errorf("text section not aligned to 2048 bytes")
	}

	b = b[2048:]
	if int64(len(b)) != int64(size) {
		return nil, fmt.Errorf("text size does not match buffer size")
	}

	return []Section{
		{"text", addr, b},
	}, nil
}
