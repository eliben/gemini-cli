package tableloader

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var csvSample1 = `id,name,age
1,joe, 29
2,rohard, 52
9,"mary lou, senior",40
`

var tsvSample1 = `id	path
20	/tmp
99	/home
`

var tsvSample2 = `x	y	a	b	c
1	2	3	4	5
i	i	i	i	i
`

var jsonSample1 = `[{"id": 20, "name": "joe"}, {"id": 44, "name": "ma"}]`

var jsonLinesSample = `{"id": 99, "name": "fuhrman"}
{"fine": true, "yes": "no"}`

func TestValidInput(t *testing.T) {
	var tests = []struct {
		format     Format
		data       string
		wantFormat Format
		wantTable  Table
	}{
		{FormatCSV, csvSample1, FormatCSV,
			Table{
				Row{"id": "1", "name": "joe", "age": " 29"},
				Row{"id": "2", "name": "rohard", "age": " 52"},
				Row{"id": "9", "name": "mary lou, senior", "age": "40"},
			}},
		{FormatTSV, tsvSample1, FormatTSV,
			Table{
				Row{"id": "20", "path": "/tmp"},
				Row{"id": "99", "path": "/home"},
			}},
		{FormatTSV, tsvSample2, FormatTSV,
			Table{
				Row{"a": "3", "b": "4", "c": "5", "x": "1", "y": "2"},
				Row{"a": "i", "b": "i", "c": "i", "x": "i", "y": "i"},
			}},
		{FormatJSON, jsonSample1, FormatJSON,
			Table{
				Row{"id": "20", "name": "joe"},
				Row{"id": "44", "name": "ma"},
			}},
		{FormatJSONLines, jsonLinesSample, FormatJSONLines,
			Table{
				Row{"id": "99", "name": "fuhrman"},
				Row{"fine": "true", "yes": "no"},
			}},
	}

	for _, tt := range tests {
		// We run each test twice: once with the format specified in tests, and
		// once with FormatUnknown.
		tfun := func(t *testing.T, format Format) {
			r := bytes.NewReader([]byte(tt.data))
			outFormat, tab, err := LoadTable(r, format)
			if err != nil {
				t.Error(err)
			}
			if outFormat != tt.wantFormat {
				t.Errorf("format got %v, want %v", outFormat, tt.wantFormat)
			}
			if diff := cmp.Diff(tab, tt.wantTable); diff != "" {
				t.Errorf("table mismatch (-want +got):\n%s", diff)
			}
		}

		t.Run(fmt.Sprintf("%v-%v", tt.format, tt.data), func(t *testing.T) {
			tfun(t, tt.format)
		})
		t.Run(fmt.Sprintf("FormatUnknown-%v", tt.data), func(t *testing.T) {
			tfun(t, FormatUnknown)
		})
	}
}

func TestErrors(t *testing.T) {
	var tests = []struct {
		format    Format
		data      string
		wantError string
	}{
		{FormatUnknown, "abcde:foo:bar:xyz", "unable to auto-detect"},
		{FormatUnknown, "abcde.foo  .bar ^xyz\n", "unable to auto-detect"},
		{FormatCSV, "id,name\n10\n", "wrong number of fields"},
		{FormatTSV, "id\tname\n10\n", "wrong number of fields"},
		{FormatJSON, "abc", "looking for beginning"},
		{FormatJSON, "{abc", "looking for beginning"},
		{FormatJSON, "[{\"abc\"", "unexpected EOF"},
	}

	for _, tt := range tests {
		t.Run(tt.data, func(t *testing.T) {
			r := bytes.NewReader([]byte(tt.data))
			_, tab, err := LoadTable(r, tt.format)
			if err == nil || tab != nil {
				t.Errorf("want error and nil table")
			}

			errs := err.Error()
			m, merr := regexp.MatchString(tt.wantError, err.Error())
			if merr != nil {
				t.Error(merr)
			}
			if !m {
				t.Errorf("got error %q, want to match %q", errs, tt.wantError)
			}
		})
	}
}

func TestPlay(t *testing.T) {
	r := bytes.NewReader([]byte(`[{"id": 20, "name": "joe"}, {"id": 44, "name": "ma"}]`))
	f, tab, err := LoadTable(r, FormatUnknown)
	fmt.Println(f, tab, err)

	{
		r := bytes.NewReader([]byte(csvSample1))
		f, tab, err := LoadTable(r, FormatUnknown)
		fmt.Println(f, tab, err)
	}

	{
		r := bytes.NewReader([]byte(tsvSample1))
		f, tab, err := LoadTable(r, FormatUnknown)
		fmt.Println(f, tab, err)
	}
}
