package csvutil

import (
	"encoding"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var (
	textMarshalerType   = reflect.TypeOf(new(encoding.TextMarshaler)).Elem()
	textUnmarshalerType = reflect.TypeOf(new(encoding.TextUnmarshaler)).Elem()
	unexportedfield     = regexp.MustCompile(`^[a-z].*$`)
)

type csvField struct {
	required   bool
	fieldName  string
	fieldIndex int
	// we cache this to prevent repeated needs for reflection
	fieldType         reflect.Kind
	sliceType         reflect.Kind
	customMarshaler   bool
	customUnmarshaler bool
}

func structureFromStruct(dest interface{}) ([]csvField, error) {
	if dest == nil {
		return nil, fmt.Errorf("provided struct cannot be nil")
	}

	csvMappings := []csvField{}
	structType := reflect.TypeOf(dest)
	for i := 0; i < reflect.ValueOf(dest).NumField(); i++ {
		fieldInfo := structType.Field(i)
		tags := strings.Split(fieldInfo.Tag.Get("csv"), ",")
		if len(tags) > 2 {
			return nil, fmt.Errorf("csvutil tags should only include name [and optionally required], found %d values", len(tags))
		}
		csvFieldName := tags[0]
		if csvFieldName == "" { // for now, ignore fields w/o a name
			continue
		}
		// if a field does have a csv tag, we must be able to access it
		if unexportedfield.MatchString(fieldInfo.Name) {
			return nil, fmt.Errorf("cannot access field '%s'", fieldInfo.Name)
		}

		if len(tags) == 2 && tags[1] != "required" {
			return nil, fmt.Errorf("unknown second value found in csv tags: '%s'", tags[1])
		}
		requiredField := len(tags) == 2

		field := csvField{
			required:   requiredField,
			fieldName:  csvFieldName,
			fieldIndex: i,
		}

		if fieldInfo.Type.Implements(textMarshalerType) {
			field.customMarshaler = true
		}
		if fieldInfo.Type.Implements(textUnmarshalerType) {
			field.customUnmarshaler = true
		}

		fieldType := fieldInfo.Type
		switch fieldType.Kind() {
		case reflect.Invalid:
			return nil, fmt.Errorf("got invalid type: %s", fieldInfo)
		case reflect.String:
			field.fieldType = reflect.String
		case reflect.Int:
			field.fieldType = reflect.Int
		case reflect.Bool:
			field.fieldType = reflect.Bool
		case reflect.Slice:
			field.fieldType = reflect.Slice
			switch fieldInfo.Type.Elem().Kind() {
			case reflect.String:
				field.sliceType = reflect.String
			case reflect.Int:
				field.sliceType = reflect.Int
			default:
				return nil, fmt.Errorf("only string & int slices allowed")
			}
		default:
			// NOTE: whether or not a marshaler type is implemented for all unknown types will
			// be audited by the NewEncoder/NewDecoder functions.
			field.fieldType = reflect.Invalid
		}

		for _, m := range csvMappings {
			if m.fieldName == field.fieldName {
				return nil, fmt.Errorf("two attributes w/ csv field name: '%s'", field.fieldName)
			}
		}

		csvMappings = append(csvMappings, field)
	}
	if len(csvMappings) == 0 {
		return nil, fmt.Errorf("no fields found for CSV marshaling")
	}

	return csvMappings, nil
}
