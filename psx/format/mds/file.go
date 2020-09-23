package mds

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type File struct {
	LeadIn       []uint32
	SubChanMixed uint8
	SubChanRaw   uint8
	PregapOffset bool
	Tracks       []Track
}

type Track struct {
	Type        uint8
	Start       [3]uint8
	ExtraOffset uint32
	StartOffset uint32
	Pregap      uint32
	Length      uint32
}

func Decode(r io.ReadSeeker) (*File, error) {
	f := &File{}
	b := binReader{r: r}

	var sig uint32
	b.Read(&sig)
	if sig != 0x4944454d {
		return nil, fmt.Errorf("not a mds file, got magic %x", sig)
	}

	var offset uint32
	b.Seek(0x50, os.SEEK_SET)
	b.Read(&offset)
	offset += 14

	var numtracks uint16
	b.Seek(int64(offset), os.SEEK_SET)
	b.Read(&numtracks)

	b.Seek(4, os.SEEK_CUR)
	b.Read(&offset)

	var c byte
	for {
		b.Read(&c)
		if c < 0xa0 {
			break
		}
		offset += 0x50
	}

	b.Seek(int64(offset)+1, os.SEEK_SET)
	b.Read(&f.SubChanMixed)
	f.SubChanRaw = f.SubChanMixed

	for i := uint16(0); i < numtracks; i++ {
		var t Track
		var extraOffset uint32

		b.Seek(int64(offset), os.SEEK_SET)
		b.Read(&t.Type)

		b.Seek(8, os.SEEK_CUR)
		b.Read(t.Start)

		b.Read(&extraOffset)

		b.Seek(int64(offset)+0x28, os.SEEK_SET)
		b.Read(&t.StartOffset)

		var gap uint32
		b.Seek(int64(extraOffset), os.SEEK_SET)
		b.Read(&gap)

		if gap != 0 && i > 0 {
			f.PregapOffset = true
		}

		f.Tracks = append(f.Tracks, t)
	}

	if b.err != nil {
		return nil, b.err
	}

	return f, nil
}

type binReader struct {
	r   io.ReadSeeker
	err error
}

func (b *binReader) Read(x interface{}) {
	if b.err != nil {
		return
	}
	b.err = binary.Read(b.r, binary.LittleEndian, x)
}

func (b *binReader) Seek(off int64, whence int) int64 {
	if b.err != nil {
		return -1
	}

	var n int64
	n, b.err = b.r.Seek(off, whence)
	return n
}
