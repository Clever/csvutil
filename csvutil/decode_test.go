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
			msg: "simple struct w/ two fields, and extraneous CSV headers",
			s: struct {
				StrField     string `csv:"string"`
				IntegerField int    `csv:"integer,required"`
			}{},
			csvFile:    "string,c1,c2,integer\ntest,1\ntest,2",
			fieldOrder: []string{"string", "integer"},
		},
		{
			msg: "simple struct w/ two fields, order opposite of headers",
			s: struct {
				IntegerField int    `csv:"integer,required"`
				StrField     string `csv:"string"`
			}{},
			csvFile:    "string,integer\ntest,1\ntest,2",
			fieldOrder: []string{"integer", "string"},
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
