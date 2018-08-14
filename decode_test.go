package csvutil

import (
	"encoding/csv"
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
			msg: "support case insensetive",
			s: struct {
				StrField     string `csv:"String"`
				IntegerField int    `csv:"integer,required"`
			}{},
			csvFile:    "strinG,integer\ntest,1\ntest,2",
			fieldOrder: []string{"String", "integer"},
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
			err: fmt.Errorf("unsupported field type found that does not implement the " +
				"encoding.TextUnmarshaler interface: custom"),
		},
	}

	for _, s := range specs {
		t.Run(s.msg, func(t *testing.T) {
			d, err := NewDecoder(strings.NewReader(s.csvFile), s.s)
			if assert.Equal(t, s.err, err, s.msg) && s.err == nil {
				fields := make([]string, len(d.mappings))
				for i, m := range d.mappings {
					fields[i] = m.fieldName
				}
				assert.Equal(t, s.fieldOrder, fields, s.msg)
			}
		})
	}
}

func TestNewDecoderFromCSVReader(t *testing.T) {
	csvStr := "field1\tfield2\nvalue1\tvalue2"
	type S struct {
		Field1 string `csv:"field1"`
		Field2 string `csv:"field2"`
	}
	r := csv.NewReader(strings.NewReader(csvStr))
	r.Comma = '\t'
	d, err := NewDecoderFromCSVReader(r, S{})
	assert.Nil(t, err)
	var s S
	err = d.Read(&s)
	assert.Nil(t, err)
	assert.Equal(t, s.Field1, "value1")
	assert.Equal(t, s.Field2, "value2")
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

type CustomTime struct {
	time.Time
}

func (t *CustomTime) UnmarshalText(b []byte) error {
	parsedTime, err := time.Parse("01/02/2006", string(b))
	if err != nil {
		return err
	}
	*t = CustomTime{parsedTime}
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
			msg: "int & string columns - case insensitive",
			res: S{
				StrField: "test",
				IntField: 1,
			},
			csvFile: "integer,StRiNg\n1,test\n2,hi",
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
			err:     errors.New("failed to coerce value 'a' (indexed 1) into integer for field intarray: strconv.Atoi: parsing \"a\": invalid syntax"),
		},
		{
			msg:     "time: custom unmarshaler type",
			csvFile: fmt.Sprintf("time\n%s\n", defaultTimeStr),
			res: S{
				Time: &defaultTime,
			},
		},
		{
			msg:     "time: optional pointer field gets nil for empty string",
			csvFile: "time,string\n,x\n",
			res: S{
				Time:     nil,
				StrField: "x",
			},
		},
		{
			msg: "ignore whitespace",
			res: S{
				StrField:  "test",
				IntField:  1,
				BoolField: true,
			},
			csvFile: "integer,string  ,  boolean \n1,test , true  \n 2,   hi,   false\n",
		},
		{
			msg: "trim whitespace array",
			res: S{
				IntArray: []int{1, 2},
			},
			csvFile: "intarray\n\" 1,2 \"\n",
		},
		{
			msg:     "trim whitespace time",
			csvFile: fmt.Sprintf("time\n  %s  \n", defaultTimeStr),
			res: S{
				Time: &defaultTime,
			},
		},
	}

	for _, s := range specs {
		t.Run(s.msg, func(t *testing.T) {
			d, err := NewDecoder(strings.NewReader(s.csvFile), S{})
			assert.Nil(t, err, s.msg)
			var val S
			err = d.Read(&val)
			if assert.Equal(t, s.err, err, s.msg) && s.err == nil {
				assert.Equal(t, s.res, val, s.msg)
			}
		})
	}
}

func TestDecoderReadRequiredMissing(t *testing.T) {
	type S struct {
		StrField string     `csv:"string,required"`
		PtrField *time.Time `csv:"ptr,required"`
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
			csvFile: "string,extra,ptr\n,,test1\ntest2,,test3", // missing in first row!
			err:     fmt.Errorf("column string required but no value found"),
		},
		{
			msg:     "required ptr column missing value",
			s:       S{},
			csvFile: "ptr,string\n,test1\ntest2,test3", // missing in first row!
			err:     fmt.Errorf("column ptr required but no value found"),
		},
	}

	for _, s := range specs {
		t.Run(s.msg, func(t *testing.T) {
			d, err := NewDecoder(strings.NewReader(s.csvFile), s.s)
			assert.Nil(t, err, s.msg)
			var val S
			err = d.Read(&val)
			assert.Equal(t, s.err, err, s.msg)
			assert.Equal(t, s.s, val, s.msg)
		})
	}
}

func TestDecodeWithPointerUnmarshal(t *testing.T) {
	type S struct {
		T CustomTime `csv:"m"`
	}
	const timeString = "10/05/2015"
	var timeValue = time.Date(2015, 10, 5, 0, 0, 0, 0, time.UTC)

	input := fmt.Sprintf("m\n%s", timeString)
	d, err := NewDecoder(strings.NewReader(input), S{})
	assert.NoError(t, err)
	var s S
	err = d.Read(&s)
	assert.NoError(t, err)
	assert.Equal(t, CustomTime{timeValue}, s.T)
}

func TestDecodeMultipleSingleColumnRows(t *testing.T) {
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

func TestDecodeMultipleMultiColumnRows(t *testing.T) {
	type MultiColumn struct {
		FieldA string `csv:"field_a"`
		FieldB string `csv:"field_b"`
	}
	expectedValues := []MultiColumn{{"x", "x"}, {"x", ""}, {"", "x"}, {"", ""}}
	input := (`field_a,field_b
x,x
x,
,x
,`)

	d, err := NewDecoder(strings.NewReader(input), MultiColumn{})
	assert.NoError(t, err)

	var actual MultiColumn // Note the same variable is being reused
	for i, expected := range expectedValues {
		assert.NoError(t, d.Read(&actual), "reading %d row", i)
		assert.Equal(t, expected.FieldA, actual.FieldA)
		assert.Equal(t, expected.FieldB, actual.FieldB)
	}

	assert.Equal(t, io.EOF, d.Read(&actual), "should return EOF at end of buffer")
}

func TestDecoderMatchedHeaders(t *testing.T) {
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
		headers []string
		csvFile string
		err     error
	}{
		{
			msg:     "no matched headers",
			headers: []string{},
			csvFile: "no_match,no_match_two\n",
		},
		{
			msg:     "one matched header",
			headers: []string{"integer"},
			csvFile: "integer,unmatched_header\n",
		},
		{
			msg:     "all matched headers",
			headers: []string{"time", "intarray", "array", "boolean", "integer", "string"},
			csvFile: "time,intarray,array,no_match,boolean,integer,string\n",
		},
		{
			msg:     "ignore whitespace in headers",
			headers: []string{"time", "intarray", "array", "boolean", "integer", "string"},
			csvFile: "  time ,  intarray,array ,no_match, boolean,integer,string \n",
		},
		{
			msg:     "ignore non-ascii characters in headers",
			headers: []string{"time", "intarray", "array", "boolean", "integer", "string"},
			csvFile: "\300time,intarray,array,boolean,integer,string\n",
		},
	}

	for _, s := range specs {
		t.Run(s.msg, func(t *testing.T) {
			d, err := NewDecoder(strings.NewReader(s.csvFile), S{})
			assert.Nil(t, err, s.msg)
			matchedHeaders := d.MatchedHeaders()
			assert.Equal(t, s.headers, matchedHeaders, s.msg)
		})
	}
}
