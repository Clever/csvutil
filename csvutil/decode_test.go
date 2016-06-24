package csvutil

import (
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
				StrField int `csv:"string"`
			}{},
			csvFile:    "string\ntest\ntest2",
			fieldOrder: []string{"string"},
		},
	}

	for _, s := range specs {
		d, err := NewDecoder(strings.NewReader(s.csvFile), s.s)
		if assert.Equal(t, s.err, err) {
			fields := make([]string, d.numColumns)
			for i, m := range d.mappings {
				fields[i] = m.fieldName
			}
			assert.Equal(t, s.fieldOrder, fields)
		}
	}
}
