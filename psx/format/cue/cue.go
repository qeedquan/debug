package cue

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/scanner"
)

type Sheet struct {
	Catalog    string
	CDTextFile string
	Flags      string
	ISRC       string
	Title      string
	Performer  string
	SongWriter string
	Files      []File
}

type File struct {
	Name   string
	Type   string
	Tracks []Track
}

type Track struct {
	Title      string
	Performer  string
	SongWriter string
	Num        int
	Audio      bool
	Mode       int
	SectorSize int
	Pregap     [3]uint8
	Postgap    [3]uint8
	Index      []Index
}

type Index struct {
	Num   int
	Start [3]uint8
}

func Parse(r io.Reader) (*Sheet, error) {
	sheet := &Sheet{}
	b := bufio.NewScanner(r)

	in := ' '
	saw := 0
	for line := 1; b.Scan(); line++ {
		var scan scanner.Scanner
		text := strings.TrimSpace(b.Text())
		scan.Init(strings.NewReader(text))
		scan.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanRawStrings | scanner.ScanInts

		scan.Scan()
		switch text := scan.TokenText(); text {
		case "REM": // ignore

		case "CATALOG", "CDTEXTFILE", "FLAGS", "ISRC":
			scan.Scan()
			if in != ' ' {
				return nil, fmt.Errorf("%d: encountered %s outside of valid context")
			}
			switch text {
			case "CATALOG":
				sheet.Catalog = text
			case "CDTEXTFILE":
				sheet.CDTextFile = text
			case "FLAGS":
				sheet.Flags = text
			case "ISRC":
				sheet.ISRC = text
			}

		case "TITLE", "PERFORMER", "SONGWRITER":
			scan.Scan()
			switch in {
			case ' ':
				sheet.Title = scan.TokenText()
			case 't':
				f := &sheet.Files[len(sheet.Files)-1]
				t := &f.Tracks[len(f.Tracks)-1]
				switch text {
				case "TITLE":
					t.Title = scan.TokenText()
				case "PERFORMER":
					t.Performer = scan.TokenText()
				case "SONGWRITER":
					t.SongWriter = scan.TokenText()
				}
			default:
				return nil, fmt.Errorf("%d: encountered %s outside of valid context")
			}

		case "TRACK":
			if in != 'f' {
				return nil, fmt.Errorf("%d: encountered %s outside of file", line)
			}

			saw = 0
			in = 't'
			scan.Scan()
			num, err := strconv.Atoi(scan.TokenText())
			if err != nil {
				return nil, fmt.Errorf("%d: %v", line, err)
			}

			audio, mode, sectorSize, err := parseTrack(&scan)
			if err != nil {
				return nil, fmt.Errorf("%d: %v", line, err)
			}

			f := &sheet.Files[len(sheet.Files)-1]
			f.Tracks = append(f.Tracks, Track{
				Num:        num,
				Audio:      audio,
				Mode:       mode,
				SectorSize: sectorSize,
			})

		case "INDEX":
			if in != 't' {
				return nil, fmt.Errorf("%d: encountered INDEX outside of TRACK", line)
			}

			saw |= 0x4
			scan.Scan()
			num, err := strconv.Atoi(scan.TokenText())
			if err != nil {
				return nil, fmt.Errorf("%d: %v", line, err)
			}

			start, err := parseTime(&scan)
			if err != nil {
				return nil, fmt.Errorf("%d: %v", line, err)
			}

			f := &sheet.Files[len(sheet.Files)-1]
			t := &f.Tracks[len(f.Tracks)-1]

			t.Index = append(t.Index, Index{
				Num:   num,
				Start: start,
			})

		case "PREGAP", "POSTGAP":
			if in != 't' {
				return nil, fmt.Errorf("%d: encountered index outside of track", line)
			}

			if saw&0x4 != 0 {
				return nil, fmt.Errorf("%d: %s must be placed before INDEX", text)
			}

			gap, err := parseTime(&scan)
			if err != nil {
				return nil, fmt.Errorf("%d: %v", line, err)
			}

			f := &sheet.Files[len(sheet.Files)-1]
			t := &f.Tracks[len(f.Tracks)-1]
			if scan.TokenText() == "PREGAP" {
				if saw&0x1 == 0 {
					t.Pregap = gap
				} else {
					return nil, fmt.Errorf("%d: only 1 %s per track", line, text)
				}
				saw |= 0x1
			} else {
				if saw&0x2 == 0 {
					t.Postgap = gap
				} else {
					return nil, fmt.Errorf("%d: only 1 %s per track", line, text)
				}
				saw |= 0x2
			}

		case "FILE":
			saw = 0
			in = 'f'
			scan.Scan()
			name, _ := strconv.Unquote(scan.TokenText())

			scan.Scan()
			typ := scan.TokenText()

			sheet.Files = append(sheet.Files, File{
				Name: name,
				Type: typ,
			})
		}
	}

	return sheet, nil
}

func parseTime(scan *scanner.Scanner) ([3]uint8, error) {
	var t [3]uint8
	for i := range t {
		scan.Scan()
		n, err := strconv.ParseInt(scan.TokenText(), 10, 8)
		if err != nil {
			return t, err
		}

		if !(0 <= n && n <= 60) {
			return t, fmt.Errorf("invalid time")
		}
		scan.Scan()

		t[i] = uint8(n)
	}

	return t, nil
}

func parseTrack(scan *scanner.Scanner) (audio bool, mode, sectorSize int, err error) {
	scan.Scan()
	text := scan.TokenText()
	if strings.HasPrefix(text, "AUDIO") {
		audio = true
		return
	}

	if !strings.HasPrefix(text, "MODE") {
		err = fmt.Errorf("invalid mode")
		return
	}

	mode, err = strconv.Atoi(text[4:])
	if err != nil {
		return
	}

	scan.Scan()

	scan.Scan()
	text = scan.TokenText()
	sectorSize, err = strconv.Atoi(text)
	if err != nil {
		return
	}

	return
}

func Serialize(s *Sheet, w io.Writer) {
	if s.Title != "" {
		fmt.Fprintln(w, s.Title)
	}

	for _, f := range s.Files {
		fmt.Fprintf(w, "FILE %q %s\n", f.Name, f.Type)
		for _, t := range f.Tracks {
			fmt.Fprintf(w, "  TRACK %02d MODE%d/%d\n", t.Num, t.Mode, t.SectorSize)
			for _, i := range t.Index {
				fmt.Fprintf(w, "    INDEX %02d %02d:%02d:%02d\n", i.Num, i.Start[0], i.Start[1], i.Start[2])
			}
		}
	}
}
