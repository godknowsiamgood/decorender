package layout

import (
	"fmt"
	"github.com/expr-lang/expr"
	"reflect"
	"strconv"
	"strings"
)

// evaluate runs the expression form of a property and hands back what it
// evaluated to, untouched. Properties that need text use replaceWithValues on
// top of this; forEach needs the value itself.
//
// A nil result comes back as an empty slice, so a node repeating over a list
// that is not there is simply not drawn.
func evaluate(str string, value any, parentValue any, valueIndex int, cache *Cache) (any, error) {
	program, err := cache.program(strings.TrimLeft(str, "~"))
	if err != nil {
		return nil, err
	}

	var result any
	if parentValue == nil {
		result, err = expr.Run(program, value)
	} else {
		result, err = expr.Run(program, map[string]any{"value": value, "parent": parentValue, "index": valueIndex})
	}
	if err != nil {
		return nil, err
	}

	if result == nil {
		return []any{}, nil
	}

	return result, nil
}

// IsExpression reports whether a property value is a template rather than a
// literal.
func IsExpression(str string) bool {
	return strings.HasPrefix(str, "~")
}

func replaceWithValues(str string, value any, parentValue any, valueIndex int, cache *Cache) (string, error) {
	if !strings.HasPrefix(str, "~") {
		return str, nil
	}

	str = strings.TrimLeft(str, "~")

	program, err := cache.program(str)
	if err != nil {
		return str, err
	}

	var result any
	if parentValue == nil {
		result, err = expr.Run(program, value)
	} else {
		result, err = expr.Run(program, map[string]any{"value": value, "parent": parentValue, "index": valueIndex})
	}
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

// RunForEach invokes cb once per element of the collection a node repeats
// over, or exactly once when it repeats over nothing.
//
// source is what the forEach property holds. A plain string names a field of
// the current data, or is a count, which is what the property has always
// meant. An expression instead hands over its result: a list to walk, a count
// to repeat, or a flag deciding whether the node is drawn at all - the way to
// say that a node belongs in one panel and not in the others.
//
// index is the caller's current iteration counter, and is what cb receives
// when nothing is repeated. A node without a forEach of its own is still
// inside whatever iteration an ancestor started, so reporting 0 there would
// make `index` collapse to the first element on every descendant.
func RunForEach(parentValue interface{}, source any, index int, cb func(value any, parentValue any, index int) error) error {
	switch v := source.(type) {
	case string:
		return runForEachNamed(parentValue, v, index, cb)
	case bool:
		if !v {
			return nil
		}
		return cb(parentValue, nil, index)
	}

	val := reflect.ValueOf(source)
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		return runForEachValues(val, cb)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return runForEachTimes(int(val.Int()), parentValue, cb)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return runForEachTimes(int(val.Uint()), parentValue, cb)
	case reflect.Float32, reflect.Float64:
		return runForEachTimes(int(val.Float()), parentValue, cb)
	default:
		return fmt.Errorf("forEach: expected a list, a count or a flag, got %T", source)
	}
}

// runForEachNamed handles the string form: a count, or the name of a field of
// the current data.
func runForEachNamed(parentValue any, name string, index int, cb func(value any, parentValue any, index int) error) error {
	if name == "" {
		return cb(parentValue, nil, index)
	}

	if num, err := strconv.Atoi(name); err == nil {
		return runForEachTimes(num, parentValue, cb)
	}

	val := reflect.ValueOf(parentValue)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	var fieldVal reflect.Value

	if val.Kind() == reflect.Struct {
		fieldVal = structField(val, name)
	} else if val.Kind() == reflect.Map {
		fieldVal = val.MapIndex(reflect.ValueOf(name))
		if !fieldVal.IsValid() {
			return fmt.Errorf("forEach: key '%v' does not exist in the map %v", name, val)
		}
		fieldVal = fieldVal.Elem()
	} else {
		return fmt.Errorf("forEach: for field '%v' the provided interface is not a map or struct <%v>", name, val)
	}

	if fieldVal.IsValid() && fieldVal.Kind() == reflect.Slice {
		return runForEachValues(fieldVal, cb)
	}

	return fmt.Errorf("forEach: specified field '%v' is not a slice or does not exist in %v", name, val)
}

func runForEachValues(val reflect.Value, cb func(value any, parentValue any, index int) error) error {
	for i := val.Len() - 1; i >= 0; i-- {
		if err := cb(val.Index(i).Interface(), val.Interface(), i); err != nil {
			return err
		}
	}
	return nil
}

func runForEachTimes(times int, parentValue any, cb func(value any, parentValue any, index int) error) error {
	for i := times - 1; i >= 0; i-- {
		if err := cb(i, parentValue, i); err != nil {
			return err
		}
	}
	return nil
}

// structField looks a field up the way expr does, so that a name means the
// same thing in a forEach and in an expression: an `expr` tag renames the
// field and "-" hides it, and an untagged field answers to its own name.
func structField(val reflect.Value, name string) reflect.Value {
	t := val.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		switch field.Tag.Get("expr") {
		case "-":
			continue
		case name:
			return val.Field(i)
		case "":
			if field.Name == name {
				return val.Field(i)
			}
		}
	}

	return reflect.Value{}
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
