package utils

import (
	"fmt"
	"github.com/antonmedv/expr"
	"reflect"
	"strconv"
	"strings"
)

func ReplaceWithValues(str string, value any, parentValue any) string {
	if !strings.HasPrefix(str, "~") {
		return str
	}

	str = strings.TrimLeft(str, "~")

	result, err := expr.Eval(str, map[string]any{"value": value, "parent": parentValue})

	if err != nil {
		return fmt.Sprintf("%v", err)
	}

	switch result.(type) {
	case string:
		return result.(string)
	default:
		return fmt.Sprintf("%v", result)
	}
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
		for i := fieldVal.Len() - 1; i >= 0; i-- {
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
