package tableloader

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Format is the type of data we're expected to load into a table. The supported
// formats are: CSV (comma-separated values), TSV (tab-separated values), JSON
// (a JSON array of objects) and "line per JSON" which has a JSON object on each
// line.
// FormatUnknown means that [LoadTable] will attempt to auto-detect the
// format based on the first bytes of the input.
type Format int

const (
	FormatUnknown = iota
	FormatCSV
	FormatTSV
	FormatJSON
	FormatJSONLines
)

func (f Format) String() string {
	switch f {
	case FormatUnknown:
		return "FormatUnknown"
	case FormatCSV:
		return "FormatCSV"
	case FormatTSV:
		return "FormatTSV"
	case FormatJSON:
		return "FormatJSON"
	case FormatJSONLines:
		return "FormatJSONLines"
	default:
		panic("unknown format")
	}
}

// Table is the data format loaded by [LoadTable]. Each row in the input
// is represented by a map from column name to column values. For example,
// for CSV data:
//
//	id,name
//	1,john
//	2,mary
//
// The loaded table will be a slice of two maps:
//
//	[0]: {"id": "1", "name": "john"}
//	[1]: {"id": "2", "name": "mary"}
//
// For JSON data, the translation is more direct as [LoadTable] expects
// JSON representing an array of objects which maps directly to this type.
type Table = []Row

type Row = map[string]string

// LoadTable loads a table from the given reader, with the given format.
// If format is [FormatUnknown], it attempts to auto-detect the format by
// peeking at the first few bytes of the input.
// It returns the detected format (or just the parameter format if it's not
// unknown), the loaded data and an error.
func LoadTable(r io.Reader, format Format) (Format, Table, error) {
	br := bufio.NewReader(r)
	if format == FormatUnknown {
		preview, err := br.Peek(512)
		if err != nil && err != io.EOF {
			return format, nil, err
		}

		if bytes.HasPrefix(bytes.TrimSpace(preview), []byte("[")) {
			format = FormatJSON
		} else if bytes.HasPrefix(bytes.TrimSpace(preview), []byte("{")) {
			format = FormatJSONLines
		} else {
			firstLine, _, found := bytes.Cut(preview, []byte("\n"))
			if !found {
				return format, nil, errors.New("unable to auto-detect table format: no newline in first line")
			}

			if bytes.IndexRune(firstLine, '\t') > 0 {
				format = FormatTSV
			} else if bytes.IndexRune(firstLine, ',') > 0 {
				format = FormatCSV
			} else {
				return format, nil, errors.New("unable to auto-detect table format from first line")
			}
		}
	}

	// Here format is known
	switch format {
	case FormatJSON:
		return loadFromJSON(br, format)
	case FormatJSONLines:
		return loadFromJSONLines(br, format)
	case FormatCSV:
		fallthrough
	case FormatTSV:
		return loadFromDelimeterSeparated(br, format)
	default:
		panic("format should be known here")
	}

	return format, nil, nil
}

func loadFromDelimeterSeparated(r io.Reader, format Format) (Format, Table, error) {
	cr := csv.NewReader(r)
	switch format {
	case FormatCSV:
		cr.Comma = ','
	case FormatTSV:
		cr.Comma = '\t'
	default:
		panic("expect FormatCSV or FormatTSV")
	}

	colNames, err := cr.Read()
	if err != nil {
		return format, nil, err
	}

	var result Table
	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return format, nil, err
		}

		// Read row into a map from column name to value.
		// csv.Reader validates the number of fields in a row, so we can assume
		// it's equal to len(colNames).
		rowMap := make(Row)
		for i, col := range row {
			rowMap[colNames[i]] = col
		}
		result = append(result, rowMap)
	}

	return format, result, err
}

func loadFromJSONLines(r io.Reader, format Format) (Format, Table, error) {
	var result Table

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var obj map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &obj); err != nil {
			return format, result, err
		}

		resultItem := make(Row)
		for k, v := range obj {
			resultItem[k] = fmt.Sprint(v)
		}
		result = append(result, resultItem)
	}
	if err := scanner.Err(); err != nil {
		return format, nil, err
	}

	return format, result, nil
}

func loadFromJSON(r io.Reader, format Format) (Format, Table, error) {
	dec := json.NewDecoder(r)

	// The JSON input will likely have numbers and other non-string types
	// unquoted, and the json package will want to decode these into appropriate
	// Go types. But we want everything in strings - so we let json decode it
	// into a map of `any`, and then convert the values to strings.
	var decoded []map[string]any
	err := dec.Decode(&decoded)
	if err != nil {
		return format, nil, err
	}

	var result Table
	for _, item := range decoded {
		resultItem := make(Row)

		for k, v := range item {
			resultItem[k] = fmt.Sprint(v)
		}
		result = append(result, resultItem)
	}
	return format, result, nil
}
