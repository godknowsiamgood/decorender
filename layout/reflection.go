package layout

import (
	"fmt"
	"github.com/antonmedv/expr"
	"github.com/godknowsiamgood/decorender/utils"
	"reflect"
	"strconv"
	"strings"
)

func replaceWithValuesUnsafe(str string, value any, parentValue any, valueIndex int, cache *Cache) string {
	v, _ := replaceWithValues(str, value, parentValue, valueIndex, cache)
	return v
}

func replaceWithValues(str string, value any, parentValue any, valueIndex int, cache *Cache) (string, error) {
	if !strings.HasPrefix(str, "~") {
		return str, nil
	}

	str = strings.TrimLeft(str, "~")

	cache.programsMx.Lock()
	defer cache.programsMx.Unlock()

	program, _ := cache.programs[utils.HashDJB2(str)]
	if program == nil {
		var err error
		program, err = expr.Compile(str)
		if err != nil {
			return str, err
		}
		cache.programs[utils.HashDJB2(str)] = program
	}

	result, err := expr.Run(program, map[string]any{"value": value, "parent": parentValue, "index": valueIndex})
	if err != nil {
		return str, err
	}

	switch result.(type) {
	case string:
		return result.(string), nil
	default:
		return fmt.Sprintf("%v", result), nil
	}
}

func RunForEach(parentValue interface{}, arrayFieldName string, cb func(value any, parentValue any, index int) error) error {
	if arrayFieldName == "" {
		return cb(parentValue, nil, 0)
	}

	if num, err := strconv.Atoi(arrayFieldName); err == nil {
		for i := num - 1; i >= 0; i-- {
			err = cb(i, parentValue, i)
			if err != nil {
				return err
			}
		}
		return nil
	}

	val := reflect.ValueOf(parentValue)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	var fieldVal reflect.Value

	if val.Kind() == reflect.Struct {
		fieldVal = val.FieldByName(arrayFieldName)
	} else if val.Kind() == reflect.Map {
		fieldVal = val.MapIndex(reflect.ValueOf(arrayFieldName))
		if !fieldVal.IsValid() {
			return fmt.Errorf("forEach: key '%v' does not exist in the map %v", arrayFieldName, val)
		}
		fieldVal = fieldVal.Elem()
	} else {
		return fmt.Errorf("forEach: for field '%v' the provided interface is not a map or struct <%v>", arrayFieldName, val)
	}

	if fieldVal.IsValid() && fieldVal.Kind() == reflect.Slice {
		for i := fieldVal.Len() - 1; i >= 0; i-- {
			if err := cb(fieldVal.Index(i).Interface(), fieldVal.Interface(), i); err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("forEach: specified field '%v' is not a slice or does not exist in %v", arrayFieldName, val)
	}

	return nil
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
