package csvutil

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type noMarshal string

type fifty string

func (f *fifty) MarshalText() ([]byte, error) {
	return []byte("50"), nil
}

func (f *fifty) UnmarshalText(text []byte) error {
	switch string(text) {
	case "50":
		*f = "50"
	case "50.0":
		*f = "50"
	case "fifty":
		*f = "50"
	}
	return nil
}

type marshalNoPointer string

func (f marshalNoPointer) MarshalText() ([]byte, error) {
	return []byte("yay"), nil
}

func (f *marshalNoPointer) UnmarshalText(text []byte) error {
	switch string(text) {
	case "yay":
		*f = "with pointer"
	}
	return nil
}

func TestStructureFromStruct(t *testing.T) {
	specs := []struct {
		msg     string
		s       interface{}
		mapping []csvField
		err     error
	}{
		{
			msg: "simple struct w/ 1 field",
			s: struct {
				IntegerField int `csv:"integer"`
			}{},
			mapping: []csvField{
				csvField{
					fieldName:  "integer",
					fieldIndex: 0,
					fieldType:  reflect.Int,
				},
			},
		},
		{
			msg: "simple struct w/ 1 field that is required",
			s: struct {
				IntegerField int `csv:"integer,required"`
			}{},
			mapping: []csvField{
				csvField{
					required:   true,
					fieldName:  "integer",
					fieldIndex: 0,
					fieldType:  reflect.Int,
				},
			},
		},
		{
			msg: "struct w/ 3 fields, 1 private",
			s: struct {
				Field1       int `csv:"f1"`
				privateField int
				Field2       string `csv:"f2"`
			}{},
			mapping: []csvField{
				csvField{
					fieldName:  "f1",
					fieldIndex: 0,
					fieldType:  reflect.Int,
				},
				csvField{
					fieldName:  "f2",
					fieldIndex: 2,
					fieldType:  reflect.String,
				},
			},
		},
		{
			msg: "struct w/ slice",
			s: struct {
				Field1 []string `csv:"f1"`
			}{},
			mapping: []csvField{
				csvField{
					fieldName:  "f1",
					fieldIndex: 0,
					fieldType:  reflect.Slice,
					sliceType:  reflect.String,
				},
			},
		},
		{
			msg: "struct w/ int slice",
			s: struct {
				Field1 []int `csv:"f1"`
			}{},
			err: nil,
			mapping: []csvField{
				csvField{
					fieldName:  "f1",
					fieldIndex: 0,
					fieldType:  reflect.Slice,
					sliceType:  reflect.Int,
				},
			},
		},
		{
			msg: "struct w/ no fields",
			s:   struct{}{},
			err: fmt.Errorf("no fields found for CSV marshaling"),
		},
		{
			msg: "struct w/ no CSV fields",
			s: struct {
				Field1 int
			}{},
			err: fmt.Errorf("no fields found for CSV marshaling"),
		},
		{
			msg: "struct w/ no CSV fields",
			s: struct {
				Field1 int
			}{},
			err: fmt.Errorf("no fields found for CSV marshaling"),
		},
		{
			msg: "struct w/ weird csv tag",
			s: struct {
				Field1 int `csv:"f1,plox-require-field"`
			}{},
			err: fmt.Errorf("unknown second value found in csv tags: 'plox-require-field'"),
		},
		{
			msg: "struct w/ repeat csv fields (f1)",
			s: struct {
				Field1       int `csv:"f1"`
				privateField int
				Field2       string `csv:"f1"`
			}{},
			err: fmt.Errorf("two attributes w/ csv field name: 'f1'"),
		},
		{
			msg: "struct w/ 3 fields, 1 private w/ csv tags",
			s: struct {
				Field1       int    `csv:"f1"`
				privateField int    `csv:"f2"`
				Field2       string `csv:"f3"`
			}{},
			err: fmt.Errorf("cannot access field 'privateField'"),
		},
		{
			msg: "struct w/ random type",
			s: struct {
				Field1 noMarshal `csv:"f1"`
			}{},
			mapping: []csvField{
				csvField{
					fieldName:  "f1",
					fieldIndex: 0,
					fieldType:  reflect.String,
				},
			},
		},
		{
			msg: "struct w/ time.Time",
			s: struct {
				Field1 *time.Time `csv:"f1"`
			}{},
			mapping: []csvField{
				csvField{
					fieldName:         "f1",
					fieldIndex:        0,
					fieldType:         reflect.Invalid,
					customMarshaler:   true,
					customUnmarshaler: true,
				},
			},
		},
		{
			msg: "struct w/ string pointer",
			s: struct {
				Field1 *string `csv:"f1"`
			}{},
			mapping: []csvField{
				csvField{
					fieldName:         "f1",
					fieldIndex:        0,
					fieldType:         reflect.Invalid,
					customMarshaler:   false,
					customUnmarshaler: false,
				},
			},
		},
		{
			msg: "struct w/ custom marshaler/unmarshaler",
			s: struct {
				Field1 marshalNoPointer `csv:"f1"`
			}{},
			mapping: []csvField{
				csvField{
					fieldName:         "f1",
					fieldIndex:        0,
					fieldType:         reflect.String,
					customMarshaler:   true,
					customUnmarshaler: true,
				},
			},
		},
		{
			msg: "struct w/ pointer custom marshaler/unmarshaler",
			s: struct {
				Field1 *fifty `csv:"f1"`
			}{},
			mapping: []csvField{
				csvField{
					fieldName:         "f1",
					fieldIndex:        0,
					fieldType:         reflect.Invalid,
					customMarshaler:   true,
					customUnmarshaler: true,
				},
			},
		},
	}

	for _, s := range specs {
		m, err := structureFromStruct(s.s)
		assert.Equal(t, s.err, err, s.msg)
		assert.Equal(t, s.mapping, m, s.msg)
	}
}
