package csvutil

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	defaultTimeStr = "2006-01-02T15:04:05Z"
)

var (
	defaultTime = time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
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
		{
			msg: "error when csv field has no UnmarshalText implementation",
			s: struct {
				StrField A `csv:"custom"`
			}{},
			csvFile: "string\n",
			err: fmt.Errorf("unsuported field type found that does not implement the " +
				"encoding.TextUnmarshaler interface: custom"),
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

type A struct{}

func (a *A) MarshalText() ([]byte, error) {
	return nil, nil
}

type M string

func (m *M) UnmarshalText(b []byte) error {
	*m = M("foo" + string(b))
	return nil
}

func TestDecoderRead(t *testing.T) {
	type S struct {
		StrField    string     `csv:"string"`
		IntField    int        `csv:"integer"`
		BoolField   bool       `csv:"boolean"`
		StringArray []string   `csv:"array"`
		IntArray    []int      `csv:"intarray"`
		Time        *time.Time `csv:"time"`
	}

	specs := []struct {
		msg     string
		res     S
		csvFile string
		err     error
	}{
		{
			msg: "string column",
			res: S{
				StrField: "test",
			},
			csvFile: "string\ntest\ntest2",
		},
		{
			msg: "int column",
			res: S{
				IntField: 1,
			},
			csvFile: "integer\n1\n2",
		},
		{
			msg: "int & string columns",
			res: S{
				StrField: "test",
				IntField: 1,
			},
			csvFile: "integer,string\n1,test\n2,hi",
		},
		{
			msg: "int & string columns flipped",
			res: S{
				StrField: "test",
				IntField: 1,
			},
			csvFile: "string,integer\ntest,1\nhi,2",
		},
		{
			msg: "bool column",
			res: S{
				BoolField: true,
			},
			csvFile: "boolean\ntrue\nt",
		},
		{
			msg: "string array",
			res: S{
				StringArray: []string{"a", "b"},
			},
			csvFile: "array\n\"a,b\"\n",
		},
		{
			msg: "int array",
			res: S{
				IntArray: []int{1, 2},
			},
			csvFile: "intarray\n\"1,2\"\n",
		},
		{
			msg:     "invalid int array",
			csvFile: "intarray\n\"1,a\"\n",
			err:     errors.New("failed to coerce value 'a' (indexed 1) into integer for field intarray: strconv.ParseInt: parsing \"a\": invalid syntax"),
		},
		{
			msg:     "time: custom unmarshaler type",
			csvFile: fmt.Sprintf("time\n%s\n", defaultTimeStr),
			res: S{
				Time: &defaultTime,
			},
		},
	}

	for _, s := range specs {
		d, err := NewDecoder(strings.NewReader(s.csvFile), S{})
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

func TestDecodeMultipleRows(t *testing.T) {
	type S struct {
		StrField string `csv:"string_column"`
	}

	expectedValues := []string{"a", "b", "c"}
	input := (`string_column
a
b
c`)

	d, err := NewDecoder(strings.NewReader(input), S{})
	assert.NoError(t, err)
	for i, val := range expectedValues {
		var s S
		assert.NoError(t, d.Read(&s), "reading %d row", i)
		assert.Equal(t, val, s.StrField)
	}

	var s S
	assert.Equal(t, io.EOF, d.Read(&s), "should return EOF at end of buffer")
}
