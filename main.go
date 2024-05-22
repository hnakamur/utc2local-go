package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/debug"
	"time"
)

const dateTimeScondLen = len("yyyy-mm-ddTHH:MM:SS")

var digTbl = buildDigitTable()

func main() {
	tz := flag.String("tz", "Z", "UTF timezone string to search")
	onlyFirst := flag.Bool("only-first", false, "convert only the first datetime in each line")
	showVersion := flag.Bool("version", false, "show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version())
		return
	}

	switch *tz {
	case "Z":
	case "+00:00":
	default:
		log.Fatal(`Flag -tz must be one of "Z" or "+00:00"`)
	}
	if err := run([]byte(*tz), *onlyFirst); err != nil {
		log.Fatal(err)
	}
}

func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(devel)"
	}
	return info.Main.Version
}

func buildDigitTable() (tbl [256]bool) {
	for b := '0'; b <= '9'; b++ {
		tbl[b] = true
	}
	return
}

const layout0 = "2006-01-02T15:04:05Z07:00"
const layout3 = "2006-01-02T15:04:05.999Z07:00"
const layout6 = "2006-01-02T15:04:05.999999Z07:00"
const layout9 = "2006-01-02T15:04:05.999999999Z07:00"

func run(tz []byte, onlyFirst bool) error {
	bw := bufio.NewWriter(os.Stdout)
	if err := convertDatetime(os.Stdin, bw, tz, onlyFirst,
		time.Now().Location()); err != nil {
		return err
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	return nil
}

func convertDatetime(r io.Reader, w io.Writer, tz []byte, onlyFirst bool,
	local *time.Location) error {
	br := bufio.NewReader(r)
	var output []byte
	for {
		input, err := br.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return err
		}
		for {
			start, end, subSecondDigitLen := findUTCDatetime(input, tz)
			if start == -1 {
				output = append(output, input...)
				if _, err := w.Write(output); err != nil {
					return err
				}
				output = output[:0]
				break
			}
			if start > 0 {
				output = append(output, input[:start]...)
			}
			var layout string
			switch subSecondDigitLen {
			case 0:
				layout = layout0
			case 3:
				layout = layout3
			case 6:
				layout = layout6
			case 9:
				layout = layout9
			}
			t, err := time.Parse(layout, string(input[start:end]))
			if err != nil {
				return err
			}
			output = t.In(local).AppendFormat(output, layout)
			input = input[end:]
			if onlyFirst {
				output = append(output, input...)
				if _, err := w.Write(output); err != nil {
					return err
				}
				output = output[:0]
				break
			}
		}
		if err == io.EOF {
			break
		}
	}
	return nil
}

func findUTCDatetime(p, tz []byte) (start, end, subSecondDigitLen int) {
	// search yyyy-mm-ddTHH:MM:SS{subsecond}?{tz}
	// subsecond = (empty) | .SSS | .SSSSSS | .SSSSSSSSS
	pos := 0
	for {
		tzPos := bytes.Index(p, tz)
		if tzPos == -1 {
			return -1, -1, 0
		}

		start, end, subSecondDigitLen = detectUTCDatetimeRange(p, tz, tzPos)
		if start != -1 {
			return start + pos, end + pos, subSecondDigitLen
		}

		pos += tzPos + len(tz)
		p = p[tzPos+len(tz):]
	}
}

func detectUTCDatetimeRange(p, tz []byte, tzPos int) (start, end, subSecondDigitLen int) {
	end = tzPos + len(tz)
	dotPos := bytes.LastIndexByte(p[:tzPos], '.')
	if dotPos == -1 {
		start = tzPos - dateTimeScondLen
		if start < 0 {
			return -1, -1, 0
		}
	} else {
		start = dotPos - dateTimeScondLen
		if start < 0 {
			return -1, -1, 0
		}
		subSecondDigitLen = tzPos - (dotPos + len("."))
		switch subSecondDigitLen {
		case 3:
			if !(digTbl[p[dotPos+1]] && digTbl[p[dotPos+2]] && digTbl[p[dotPos+3]]) {
				return -1, -1, 0
			}
		case 6:
			if !(digTbl[p[dotPos+1]] && digTbl[p[dotPos+2]] && digTbl[p[dotPos+3]] &&
				digTbl[p[dotPos+4]] && digTbl[p[dotPos+5]] && digTbl[p[dotPos+6]]) {
				return -1, -1, 0
			}
		case 9:
			if !(digTbl[p[dotPos+1]] && digTbl[p[dotPos+2]] && digTbl[p[dotPos+3]] &&
				digTbl[p[dotPos+4]] && digTbl[p[dotPos+5]] && digTbl[p[dotPos+6]] &&
				digTbl[p[dotPos+7]] && digTbl[p[dotPos+8]] && digTbl[p[dotPos+9]]) {
				return -1, -1, 0
			}
		default:
			return -1, -1, 0
		}
	}

	// Validate yyyy-mm-ddTHH:MM:SS part.
	if !(digTbl[p[start]] && digTbl[p[start+1]] && digTbl[p[start+2]] && digTbl[p[start+3]] &&
		p[start+4] == '-' && digTbl[p[start+5]] && digTbl[p[start+6]] &&
		p[start+7] == '-' && digTbl[p[start+8]] && digTbl[p[start+9]] &&
		p[start+10] == 'T' && digTbl[p[start+11]] && digTbl[p[start+12]] &&
		p[start+13] == ':' && digTbl[p[start+14]] && digTbl[p[start+15]] &&
		p[start+16] == ':' && digTbl[p[start+17]] && digTbl[p[start+18]]) {
		return -1, -1, 0
	}
	return start, end, subSecondDigitLen
}
