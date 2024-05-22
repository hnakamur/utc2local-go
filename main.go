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
		// log.Printf("after ReadBytes, input=%q, err=%v", input, err)
		if err != nil && err != io.EOF {
			return err
		}
		for {
			start, end, subSecondDigitLen := findUTCDatetime(input, tz)
			// log.Printf("after findUTCDatetime, start=%d, end=%d, subSecondDigitLen=%d", start, end, subSecondDigitLen)
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
			// log.Printf("after time.Parse, t=%v, err=%v", t, err)
			if err != nil {
				return err
			}
			// log.Printf("t.In(local)=%v", t.In(local))
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
	tzPos := bytes.Index(p, tz)
	// log.Printf("after bytes.Index, tzPos=%d", tzPos)
	if tzPos == -1 {
		return -1, -1, 0
	}
	end = tzPos + len(tz)
	dotPos := bytes.LastIndexByte(p[:tzPos], '.')
	// log.Printf("dotPos=%d", dotPos)
	if dotPos == -1 {
		start = tzPos - dateTimeScondLen
		// log.Printf("start#1=%d", start)
		if start < 0 {
			return -1, -1, 0
		}
	} else {
		start = dotPos - dateTimeScondLen
		// log.Printf("start#2=%d", start)
		if start < 0 {
			return -1, -1, 0
		}
		subSecondDigitLen = tzPos - (dotPos + len("."))
		// log.Printf("subSecondDigitLen=%d, subsecond=%s", subSecondDigitLen, string(p[dotPos+1:tzPos]))
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

	// log.Printf("datetimePart=%s", string(p[start:end]))
	// log.Printf("0:%v, 1:%v, 2:%v, 3:%v,\n4:%v, 5:%v, 6:%v,\n7:%v, 8:%v, 9:%v,\n10:%v, 11:%v, 12:%v,\n13:%v, 14:%v, 15:%v,\n16:%v, 17:%v, 18:%v,",
	// 	digTbl[p[start]], digTbl[p[start+1]], digTbl[p[start+2]], digTbl[p[start+3]],
	// 	p[start+4] == '-', digTbl[p[start+5]], digTbl[p[start+6]],
	// 	p[start+7] == '-', digTbl[p[start+8]], digTbl[p[start+9]],
	// 	p[start+10] == 'T', digTbl[p[start+11]], digTbl[p[start+12]],
	// 	p[start+13] == ':', digTbl[p[start+14]], digTbl[p[start+15]],
	// 	p[start+16] == ':', digTbl[p[start+17]], digTbl[p[start+18]],
	// )

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
