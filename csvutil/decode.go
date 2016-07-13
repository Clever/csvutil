package csvutil

import (
	"encoding"
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// Decoder manages reading data from a CSV into tagged structs.
type Decoder struct {
	r          *csv.Reader
	mappings   []csvField
	numColumns int
}

// NewDecoder initializes itself with the headers of the CSV file to build mappings
// to read data into structs.
func NewDecoder(r io.Reader, dest interface{}) (Decoder, error) {
	csvR := csv.NewReader(r)
	mappings, err := structureFromStruct(dest)
	if err != nil {
		return Decoder{}, err
	}

	// ensure that all "unknown" types have their own text unmarshaler
	for _, m := range mappings {
		if m.fieldType == reflect.Invalid && !m.customUnmarshaler {
			// TODO: error out?
		}
	}

	headers, err := csvR.Read()
	if err != nil {
		return Decoder{}, fmt.Errorf("failed to find headers: %s", err)
	}

	numColumns := len(headers)
	sortedMappings := make([]csvField, numColumns)
	extraHeaders := []string{} // TODO: do anything with this?
	headersSeen := map[string]bool{}
	// Sort headers in line w/ CSV columns
	for i, h := range headers {
		// ensure unique CSV headers
		if headersSeen[h] {
			return Decoder{}, fmt.Errorf("saw header column '%s' twice, CSV headers must be unique", h)
		}
		headersSeen[h] = true

		// slot field info in array parallel to CSV column
		for _, f := range mappings {
			if h == f.fieldName {
				sortedMappings[i] = f
			}
		}
		// check if field not set
		if sortedMappings[i].fieldName == "" {
			extraHeaders = append(extraHeaders, h)
		}
	}

	// Ensure that all required columns are present
	for _, f := range mappings {
		if f.required && !headersSeen[f.fieldName] {
			return Decoder{}, fmt.Errorf("column '%s' required but not found", f.fieldName)
		}
	}

	return Decoder{
		r:          csvR,
		mappings:   sortedMappings,
		numColumns: numColumns,
	}, nil
}

// Read decodes data from a CSV row into a struct. The struct must be passed as a pointer
// into Read.
func (d Decoder) Read(dest interface{}) error {
	destStruct := reflect.ValueOf(dest)
	if dest == nil {
		return fmt.Errorf("Destination struct passed in cannot be nil")
	} else if destStruct.Type().Kind() != reflect.Ptr {
		return fmt.Errorf("Destination struct passed in must be pointer")
	} else if destStruct.Elem().Kind() == reflect.Interface {
		return fmt.Errorf("Destination struct cannot be an interface")
	}

	row, err := d.r.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV row: %s", err)
	}

	if len(row) != d.numColumns {
		return fmt.Errorf("expected %d columns, found %d", d.numColumns, len(row))
	}

	// for i, m := range d.mappings {
	for i, strValue := range row {
		m := d.mappings[i]
		// skip column if we have no mapping
		if m.fieldName == "" {
			continue
		} else if m.required && strValue == "" {
			return fmt.Errorf("column %s required but no value found", m.fieldName)
		}

		if m.customUnmarshaler {
			v := destStruct.Elem().Field(m.fieldIndex)
			v.Set(reflect.New(v.Type().Elem()))

			u := v.Interface().(encoding.TextUnmarshaler)
			if err := u.UnmarshalText([]byte(strValue)); err != nil {
				return fmt.Errorf("failed to coerce value '%s' using custom marshaler for field %s: %s",
					strValue, m.fieldName, err)
			}
			continue
		}

		switch m.fieldType {
		case reflect.String:
			destStruct.Elem().Field(m.fieldIndex).SetString(strValue)
		case reflect.Int:
			intVal, err := strconv.Atoi(strValue)
			if err != nil {
				return fmt.Errorf("failed to coerce value '%s' into integer for field %s",
					strValue, m.fieldName)
			}
			destStruct.Elem().Field(m.fieldIndex).SetInt(int64(intVal))
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(strValue)
			if err != nil {
				return fmt.Errorf("failed to coerce value '%s' into boolean for field %s",
					strValue, m.fieldName)
			}
			destStruct.Elem().Field(m.fieldIndex).SetBool(boolVal)
		case reflect.Slice:
			arrayStrValues := strings.Split(strValue, ",")
			switch m.sliceType {
			case reflect.String:
				destStruct.Elem().Field(m.fieldIndex).Set(reflect.ValueOf(arrayStrValues))
			case reflect.Int:
				arrayIntValues := make([]int, len(arrayStrValues))
				for i, s := range arrayStrValues {
					intVal, err := strconv.Atoi(s)
					if err != nil {
						return fmt.Errorf("failed to coerce value '%s' (indexed %d) into integer for field %s: %s",
							s, i, m.fieldName, err)
					}
					arrayIntValues[i] = int(intVal)
				}
				destStruct.Elem().Field(m.fieldIndex).Set(reflect.ValueOf(arrayIntValues))
			default:
				panic("slice fields can only be string.")
			}
		default:
			panic(fmt.Sprintf("type not found: %s", m.fieldType))
		}
	}

	return nil
}
