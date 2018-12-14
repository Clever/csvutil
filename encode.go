package csvutil

import (
	"encoding"
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Encoder manages writing a tagged struct into a CSV
type Encoder struct {
	w        *csv.Writer
	mu       *sync.Mutex
	mappings []csvField
}

// NewEncoder prepares mappings from struct to CSV based on struct tags.
func NewEncoder(w io.Writer, dest interface{}) (Encoder, error) {
	csvW := csv.NewWriter(w)
	return NewEncoderFromCSVWriter(csvW, dest)
}

// NewEncoderFromCSVWriter intializes an encoder using the given csv.Writer.
// This allows the caller to configure options on the csv.Writer (e.g. what
// delimiter to use) instead of using the defaults.
func NewEncoderFromCSVWriter(csvW *csv.Writer, dest interface{}) (Encoder, error) {
	mappings, err := structureFromStruct(dest)
	if err != nil {
		return Encoder{}, err
	}
	defer csvW.Flush()

	// ensure that all "unknown" types have their own text marshaler
	for _, m := range mappings {
		if m.fieldType == reflect.Invalid && !m.customMarshaler {
			return Encoder{}, fmt.Errorf("unsuported field type found that does not "+
				"implement the encoding.TextMarshaler interface: %s", m.fieldName)
		}
	}

	headers := make([]string, len(mappings))
	for i, m := range mappings {
		headers[i] = m.fieldName
	}

	if err = csvW.Write(headers); err != nil {
		return Encoder{}, fmt.Errorf("failed to write headers: %s", err)
	}

	return Encoder{
		mu:       &sync.Mutex{},
		w:        csvW,
		mappings: mappings,
	}, nil
}

// Write encodes the values of a struct into a CSV row and writes to the underlying io.writer.
func (e Encoder) Write(src interface{}) error {
	srcStruct := reflect.ValueOf(src)
	if src == nil {
		return fmt.Errorf("Source struct passed in cannot be nil")
	} else if srcStruct.Type().Kind() == reflect.Ptr {
		srcStruct = srcStruct.Elem()
	}

	rowValues := make([]string, len(e.mappings))
	for i, m := range e.mappings {
		v := srcStruct.Field(m.fieldIndex)

		if m.customMarshaler {
			u := v.Interface().(encoding.TextMarshaler)
			buf, err := u.MarshalText()
			if err != nil {
				return fmt.Errorf("failed to coerce value '%s' into string using custom marshaler for field %s: %s",
					v, m.fieldName, err)
			}
			rowValues[i] = string(buf)
			continue
		}

		switch m.fieldType {
		case reflect.String:
			rowValues[i] = v.String()
		case reflect.Int:
			rowValues[i] = strconv.Itoa(int(v.Int()))
		case reflect.Bool:
			rowValues[i] = strconv.FormatBool(v.Bool())
		case reflect.Slice:
			switch m.sliceType {
			case reflect.String:
				rowValues[i] = strings.Join(v.Interface().([]string), ",")
			case reflect.Int:
				intArray := v.Interface().([]int)
				strArray := make([]string, len(intArray))
				for i, iVal := range intArray {
					strArray[i] = strconv.Itoa(iVal)
				}
				rowValues[i] = strings.Join(strArray, ",")
			default:
				panic("slice fields can only be string.")
			}
		default:
			panic(fmt.Sprintf("type not found: %s", m.fieldType))
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	defer e.w.Flush()
	return e.w.Write(rowValues)
}
