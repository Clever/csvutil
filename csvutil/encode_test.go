package csvutil

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEncoderValid(t *testing.T) {
	type valid struct {
		StrField string `csv:"string"`
		IntField int    `csv:"integer"`
	}
	var buf bytes.Buffer

	_, err := NewEncoder(&buf, valid{})
	assert.Nil(t, err)
	assert.Equal(t, "string,integer\n", buf.String())
}

func TestNewEncoderMissingField(t *testing.T) {
	type ignoredFieldStruct struct {
		StrField     string `csv:"string"`
		IgnoredField string
		IntField     int `csv:"integer"`
	}
	var buf bytes.Buffer

	_, err := NewEncoder(&buf, ignoredFieldStruct{})
	assert.Nil(t, err)
	assert.Equal(t, "string,integer\n", buf.String())
}

type B struct{}

func (b *B) UnmarshalText(buf []byte) error {
	return nil
}

func TestNewEncoderMissingMarshalerInterface(t *testing.T) {
	type ignoredFieldStruct struct {
		CustomField B `csv:"custom"`
	}
	var buf bytes.Buffer

	_, err := NewEncoder(&buf, ignoredFieldStruct{})
	assert.Equal(t, errors.New("unsuported field type found that does not implement"+
		" the encoding.TextMarshaler interface: custom"), err)
}

func TestEncodeValid(t *testing.T) {
	type valid struct {
		StrField string `csv:"string"`
		IntField int    `csv:"integer"`
	}
	x := valid{
		StrField: "foo",
		IntField: 100,
	}
	var buf bytes.Buffer

	enc, err := NewEncoder(&buf, valid{})
	assert.Nil(t, err)
	err = enc.Write(x)
	assert.Nil(t, err)
	assert.Equal(t, "string,integer\nfoo,100\n", buf.String())
}

func TestWriteWithMissingField(t *testing.T) {
	type ignoredFieldStruct struct {
		StrField     string `csv:"string"`
		IgnoredField string
		IntField     int `csv:"integer"`
	}
	x := ignoredFieldStruct{
		StrField: "foo",
		IntField: 100,
	}
	var buf bytes.Buffer

	enc, err := NewEncoder(&buf, ignoredFieldStruct{})
	assert.Nil(t, err)
	err = enc.Write(x)
	assert.Nil(t, err)
	assert.Equal(t, "string,integer\nfoo,100\n", buf.String())
}

func TestWriteCustomMarshaler(t *testing.T) {
	type ignoredFieldStruct struct {
		StrField  string     `csv:"string"`
		TimeField *time.Time `csv:"time"`
	}
	x := ignoredFieldStruct{
		StrField:  "foo",
		TimeField: &defaultTime,
	}
	var buf bytes.Buffer

	enc, err := NewEncoder(&buf, ignoredFieldStruct{})
	assert.Nil(t, err)
	err = enc.Write(x)
	assert.Nil(t, err)
	assert.Equal(t, fmt.Sprintf("string,time\nfoo,%s\n", defaultTimeStr), buf.String())

	// test passing in a pointer
	buf.Reset()
	enc, err = NewEncoder(&buf, ignoredFieldStruct{})
	assert.Nil(t, err)
	err = enc.Write(&x)
	assert.Nil(t, err)
	assert.Equal(t, fmt.Sprintf("string,time\nfoo,%s\n", defaultTimeStr), buf.String())
}

func TestWriteFailNil(t *testing.T) {
	type valid struct {
		StrField string `csv:"string"`
	}
	var buf bytes.Buffer

	enc, err := NewEncoder(&buf, valid{})
	assert.Nil(t, err)
	assert.Equal(t, enc.Write(nil), errors.New("Source struct passed in cannot be nil"))
}
