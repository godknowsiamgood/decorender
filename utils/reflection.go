package utils

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"
)

func ReplaceWithValues(str string, value any, parentValue any) string {
	result := replaceWithValues(str, value)
	result = strings.ReplaceAll(result, "{{{", "{{")
	result = strings.ReplaceAll(result, "}}}", "}}")
	return replaceWithValues(result, parentValue)
}

func replaceWithValues(str string, value any) string {
	if !strings.Contains(str, "{{") {
		return str
	}

	fm := template.FuncMap{
		"divide": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"multiply": func(a, b float64) float64 {
			return a * b
		},
		"add": func(a, b float64) float64 {
			return a + b
		},
		"sub": func(a, b float64) float64 {
			return a - b
		},
	}

	tmpl, err := template.New("template").Funcs(fm).Parse(str)
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

func RunForEach(parentValue interface{}, arrayFieldName string, cb func(value any, iteratorValue any)) {
	if arrayFieldName == "" {
		cb(parentValue, nil)
		return
	}

	if num, err := strconv.Atoi(arrayFieldName); err == nil {
		for i := 0; i < num; i++ {
			cb(i, parentValue)
		}
		return
	}

	val := reflect.ValueOf(parentValue)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		fmt.Println("The provided interface is not a struct")
		return
	}

	fieldVal := val.FieldByName(arrayFieldName)

	if fieldVal.IsValid() && fieldVal.Kind() == reflect.Slice {
		for i := 0; i < fieldVal.Len(); i++ {
			cb(fieldVal.Index(i).Interface(), fieldVal.Interface())
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
