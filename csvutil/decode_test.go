package csvutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDecoder(t *testing.T) {
	specs := []struct {
		msg        string
		s          interface{}
		csvFile    string
		fieldOrder []string
		err        error
	}{
		{
			msg: "simple struct w/ string field",
			s: struct {
				StrField string `csv:"string"`
			}{},
			csvFile:    "string\ntest\ntest2",
			fieldOrder: []string{"string"},
		},
		{
			msg: "simple struct w/ two fields",
			s: struct {
				StrField     string `csv:"string"`
				IntegerField int    `csv:"integer,required"`
			}{},
			csvFile:    "string,integer\ntest,1\ntest,2",
			fieldOrder: []string{"string", "integer"},
		},
		{
			msg: "simple struct w/ two fields, csv missing one",
			s: struct {
				StrField     string `csv:"string"`
				IntegerField int    `csv:"integer"`
			}{},
			csvFile:    "integer\n1\n2",
			fieldOrder: []string{"integer"},
		},
		{
			msg: "simple struct w/ two fields, and extraneous CSV headers",
			s: struct {
				StrField     string `csv:"string"`
				IntegerField int    `csv:"integer,required"`
			}{},
			csvFile:    "string,c1,c2,integer\ntest,1\ntest,2",
			fieldOrder: []string{"string", "", "", "integer"},
		},
		{
			msg: "simple struct w/ two fields, order opposite of headers",
			s: struct {
				IntegerField int    `csv:"integer,required"`
				StrField     string `csv:"string"`
			}{},
			csvFile:    "string,integer\ntest,1\ntest,2",
			fieldOrder: []string{"string", "integer"},
		},
		{
			msg: "error when missing required field",
			s: struct {
				IntegerField int    `csv:"integer,required"`
				StrField     string `csv:"string"`
			}{},
			csvFile: "string\ntest\ntest2",
			err:     fmt.Errorf("column 'integer' required but not found"),
		},
		{
			msg: "error when csv file has header twice",
			s: struct {
				StrField string `csv:"string"`
			}{},
			csvFile: "string,test,string\n",
			err:     fmt.Errorf("saw header column 'string' twice, CSV headers must be unique"),
		},
	}

	for _, s := range specs {
		d, err := NewDecoder(strings.NewReader(s.csvFile), s.s)
		if assert.Equal(t, s.err, err, s.msg) && s.err == nil {
			fields := make([]string, len(d.mappings))
			for i, m := range d.mappings {
				fields[i] = m.fieldName
			}
			assert.Equal(t, s.fieldOrder, fields, s.msg)
		}
	}
}

func TestDecoderRead(t *testing.T) {
	type S struct {
		StrField  string `csv:"string"`
		IntField  int    `csv:"integer"`
		BoolField bool   `csv:"boolean"`
	}

	specs := []struct {
		msg     string
		s       S
		res     S
		csvFile string
		err     error
	}{
		{
			msg: "string column",
			s:   S{},
			res: S{
				StrField: "test",
			},
			csvFile: "string\ntest\ntest2",
		},
		{
			msg: "int column",
			s:   S{},
			res: S{
				IntField: 1,
			},
			csvFile: "integer\n1\n2",
		},
		{
			msg: "int & string columns",
			s:   S{},
			res: S{
				StrField: "test",
				IntField: 1,
			},
			csvFile: "integer,string\n1,test\n2,hi",
		},
		{
			msg: "int & string columns flipped",
			s:   S{},
			res: S{
				StrField: "test",
				IntField: 1,
			},
			csvFile: "string,integer\ntest,1\nhi,2",
		},
		{
			msg: "bool column",
			s:   S{},
			res: S{
				BoolField: true,
			},
			csvFile: "boolean\ntrue\nt",
		},
	}

	for _, s := range specs {
		d, err := NewDecoder(strings.NewReader(s.csvFile), s.s)
		assert.Nil(t, err, s.msg)
		var val S
		err = d.Read(&val)
		if assert.Equal(t, s.err, err, s.msg) && s.err == nil {
			assert.Equal(t, s.res, val, s.msg)
		}
	}
}

func TestDecoderReadRequiredMissing(t *testing.T) {
	type S struct {
		StrField string `csv:"string,required"`
	}

	specs := []struct {
		msg     string
		s       S
		csvFile string
		err     error
	}{
		{
			msg:     "required string column missing value",
			s:       S{},
			csvFile: "string,extra\n,\ntest2,", // missing in first row!
			err:     fmt.Errorf("column string required but no value found"),
		},
	}

	for _, s := range specs {
		d, err := NewDecoder(strings.NewReader(s.csvFile), s.s)
		assert.Nil(t, err, s.msg)
		var val S
		err = d.Read(&val)
		assert.Equal(t, s.err, err, s.msg)
		assert.Equal(t, val, S{}, s.msg)
	}
}
