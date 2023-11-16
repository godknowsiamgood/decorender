package utils

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"text/template"
)

func ReplaceWithValues(str string, value interface{}) string {
	if !strings.Contains(str, "{{") {
		return str
	}

	tmpl, err := template.New("template").Parse(str)
	if err != nil {
		return str
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, value)
	if err != nil {
		return str
	}

	return buf.String()
}

func RunForEach(inputStruct interface{}, arrayFieldName string, cb func(value interface{})) {
	if arrayFieldName == "" {
		cb(inputStruct)
		return
	}

	// Use reflection to inspect the type of the inputStruct
	val := reflect.ValueOf(inputStruct)

	// If the inputStruct is a pointer, we need to dereference it to get the value it points to
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Ensure that we're dealing with a struct
	if val.Kind() != reflect.Struct {
		fmt.Println("The provided interface is not a struct")
		return
	}

	// Get the field by name
	fieldVal := val.FieldByName(arrayFieldName)

	// Check if the field exists and is a slice
	if fieldVal.IsValid() && fieldVal.Kind() == reflect.Slice {
		// Iterate over the slice
		for i := 0; i < fieldVal.Len(); i++ {
			// Call the callback with each element in the slice
			cb(fieldVal.Index(i).Interface())
		}
	} else {
		fmt.Println("The specified field is not a slice or does not exist")
	}
}

func ScaleAllValues(data any, scale float64) {
	scaleValue(reflect.ValueOf(data), scale)
}

func scaleValue(v reflect.Value, scale float64) {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		scaleValue(v.Elem(), scale)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			scaleValue(v.Field(i), scale)
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			scaleValue(v.Index(i), scale)
		}
	case reflect.Float64:
		if v.CanSet() {
			v.SetFloat(v.Float() * scale)
		}
	default:
	}
}
