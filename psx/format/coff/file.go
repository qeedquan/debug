package coff

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

type FileHeader struct {
	Magic       uint16
	NumSections uint16
	Timestamp   uint32
	SymOffset   uint32
	SymHdrSize  uint32
	OptHdrSize  uint16
	Flags       uint16
}

type ExecHeader struct {
	Magic     uint16
	Version   uint16
	TextSize  uint32
	DataSize  uint32
	BSSSize   uint32
	Entry     uint32
	TextStart uint32
	DataStart uint32
}

type SectionHeader struct {
	Name       [8]byte
	Paddr      uint32
	Vaddr      uint32
	Size       uint32
	ScnOffset  uint32
	RelOffset  uint32
	LnnoOffset uint32
	NumRelocs  uint32
	NumLnno    uint32
	Flags      uint32
}

type Section struct {
	SectionHeader
	Data []byte
}

type File struct {
	FileHeader
	ExecHeader
	Sections []Section
}

func Decode(r io.ReaderAt) (*File, error) {
	f := &File{}
	sr := io.NewSectionReader(r, 0, math.MaxUint32)

	err := binary.Read(sr, binary.LittleEndian, &f.FileHeader)
	if err != nil {
		return nil, err
	}

	if f.FileHeader.Magic != 0x0162 {
		return nil, fmt.Errorf("invalid magic")
	}

	err = binary.Read(sr, binary.LittleEndian, &f.ExecHeader)
	if err != nil {
		return nil, err
	}

	const sectionHeaderSize = 44
	var s Section
	for i := uint16(0); i < f.NumSections; i++ {
		off := int64(f.SymHdrSize) + int64(f.OptHdrSize) + sectionHeaderSize*int64(i)
		_, err = sr.Seek(off, os.SEEK_SET)
		if err != nil {
			return nil, err
		}

		err = binary.Read(sr, binary.LittleEndian, &s.SectionHeader)
		if err != nil {
			return nil, err
		}

		if s.ScnOffset != 0 {
			_, err = sr.Seek(int64(s.ScnOffset), os.SEEK_SET)
			if err != nil {
				return nil, err
			}

			s.Data = make([]byte, s.Size)
			_, err = io.ReadAtLeast(sr, s.Data, len(s.Data))
			if err != nil {
				return nil, err
			}
		}

		f.Sections = append(f.Sections, s)
	}

	return f, nil
}
