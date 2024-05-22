package main

import (
	"strings"
	"testing"
	"time"
)

func BenchmarkConvertDatetime(b *testing.B) {
	local, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		b.Fatal(err)
	}

	input := "time:2024-05-22T12:34:56.123Z\turi:/\ttime2:2024-05-22T23:34:56.123Z\n" +
		"time:2024-05-22T12:34:57.123456Z\turi:/\ttime2:2024-05-22T23:34:57.123456789Z\n"
	tz := []byte("Z")
	onlyFirst := false
	for i := 0; i < b.N; i++ {
		r := strings.NewReader(input)
		var sb strings.Builder
		if err := convertDatetime(r, &sb, tz, onlyFirst, local); err != nil {
			b.Fatal(err)
		}
	}
}

func TestConvertDatetime(t *testing.T) {
	testCases := []struct {
		input     string
		tz        string
		onlyFirst bool
		want      string
	}{
		{
			input: "time:2024-05-22T12:34:56.123Z\turi:/",
			tz:    "Z", onlyFirst: true,
			want: "time:2024-05-22T21:34:56.123+09:00\turi:/",
		},
		{
			input: "time:2024-05-22T12:34:56.123Z\turi:/\n" +
				"time:2024-05-22T23:34:56.123Z\turi:/",
			tz: "Z", onlyFirst: true,
			want: "time:2024-05-22T21:34:56.123+09:00\turi:/\n" +
				"time:2024-05-23T08:34:56.123+09:00\turi:/",
		},
		{
			input: "time:2024-05-22T12:34:56.123Z\turi:/\ttime2:2024-05-22T23:34:56Z",
			tz:    "Z", onlyFirst: false,
			want: "time:2024-05-22T21:34:56.123+09:00\turi:/\ttime2:2024-05-23T08:34:56+09:00",
		},
		{
			input: "time:2024-05-22T12:34:56.123Z\turi:/Z\ttime2:2024-05-22T23:34:56Z",
			tz:    "Z", onlyFirst: false,
			want: "time:2024-05-22T21:34:56.123+09:00\turi:/Z\ttime2:2024-05-23T08:34:56+09:00",
		},
		{
			input: "time:2024-05-22T12:34:56.123Z\turi:/\ttime2:2024-05-22T23:34:56.123Z\n" +
				"time:2024-05-22T12:34:57.123456Z\turi:/\ttime2:2024-05-22T23:34:57.123456789Z\n",
			tz: "Z", onlyFirst: false,
			want: "time:2024-05-22T21:34:56.123+09:00\turi:/\ttime2:2024-05-23T08:34:56.123+09:00\n" +
				"time:2024-05-22T21:34:57.123456+09:00\turi:/\ttime2:2024-05-23T08:34:57.123456789+09:00\n",
		},
	}
	local, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		var b strings.Builder
		r := strings.NewReader(tc.input)
		if err := convertDatetime(r, &b, []byte(tc.tz), tc.onlyFirst, local); err != nil {
			t.Fatal(err)
		}
		got := b.String()
		if got != tc.want {
			t.Errorf("result mismatch,\n got=%q,\nwant=%q,\ninput=%q, tz=%s, onlyFirst=%v",
				got, tc.want, tc.input, tc.tz, tc.onlyFirst)
		}
	}
}

func TestFindUTCDatetime(t *testing.T) {
	testCases := []struct {
		input                 string
		tz                    string
		wantStart             int
		wantEnd               int
		wantSubSecondDigitLen int
	}{
		{
			input: "time:2024-05-22T12:34:56.123Z\turi:/", tz: "Z",
			wantStart: len("time:"), wantEnd: len("time:2024-05-22T12:34:56.123Z"),
			wantSubSecondDigitLen: 3,
		},
		{
			input: "uri:/Z\ttime:2024-05-22T12:34:56.123Z", tz: "Z",
			wantStart: len("uri:/Z\ttime:"), wantEnd: len("uri:/Z\ttime:2024-05-22T12:34:56.123Z"),
			wantSubSecondDigitLen: 3,
		},
		{
			input: "uri:/2024-05-22T12:34:56.12bZ\ttime:2024-05-22T12:34:56.123Z", tz: "Z",
			wantStart: len("uri:/2024-05-22T12:34:56.12bZ\ttime:"), wantEnd: len("uri:/2024-05-22T12:34:56.12bZ\ttime:2024-05-22T12:34:56.123Z"),
			wantSubSecondDigitLen: 3,
		},
		{
			input: "uri:/+00:00\ttime:2024-05-22T12:34:56.123+00:00", tz: "+00:00",
			wantStart: len("uri:/+00:00\ttime:"), wantEnd: len("uri:/+00:00\ttime:2024-05-22T12:34:56.123+00:00"),
			wantSubSecondDigitLen: 3,
		},
		{
			input: "uri:/2024-05-22T12:34:56.12b+00:00\ttime:2024-05-22T12:34:56.123+00:00", tz: "+00:00",
			wantStart: len("uri:/2024-05-22T12:34:56.12b+00:00\ttime:"), wantEnd: len("uri:/2024-05-22T12:34:56.12b+00:00\ttime:2024-05-22T12:34:56.123+00:00"),
			wantSubSecondDigitLen: 3,
		},
	}
	for _, tc := range testCases {
		gotStart, gotEnd, gotSubSecondDigitLen := findUTCDatetime([]byte(tc.input), []byte(tc.tz))
		if gotStart != tc.wantStart || gotEnd != tc.wantEnd || gotSubSecondDigitLen != tc.wantSubSecondDigitLen {
			t.Errorf("result mismatch, gotStart=%d, wantStart=%d, gotEnd=%d, wantEnd=%d, gotSubSecondDigitLen=%d, wantSubSecondDigitLen=%d,\ninput=%q",
				gotStart, tc.wantStart, gotEnd, tc.wantEnd, gotSubSecondDigitLen, tc.wantSubSecondDigitLen, tc.input)
		}
	}
}
