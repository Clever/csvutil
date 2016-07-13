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

type Encoder struct {
	w        *csv.Writer
	mappings []csvField
}

func NewEncoder(w io.Writer, dest interface{}) (Encoder, error) {
	csvW := csv.NewWriter(w)
	mappings, err := structureFromStruct(dest)
	if err != nil {
		return Encoder{}, err
	}
	defer csvW.Flush()

	// ensure that all "unknown" types have their own text marshaler
	for _, m := range mappings {
		if m.fieldType == reflect.Invalid && !m.customMarshaler {
			// TODO: error out?
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
		w:        csvW,
		mappings: mappings,
	}, nil
}

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

	defer e.w.Flush()
	return e.w.Write(rowValues)
}
