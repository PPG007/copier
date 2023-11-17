package copier

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

var (
	TimeStringConverter = Converter{
		Origin: reflect.TypeOf(time.Time{}),
		Target: reflect.TypeOf(""),
		Fn: func(fromValue reflect.Value, toType reflect.Type) (reflect.Value, error) {
			t, ok := fromValue.Interface().(time.Time)
			if ok {
				return reflect.ValueOf(t.Format(time.RFC3339)), nil
			}
			return fromValue, nil
		},
	}

	StringTimeConverter = Converter{
		Origin: reflect.TypeOf(""),
		Target: reflect.TypeOf(time.Time{}),
		Fn: func(fromValue reflect.Value, toType reflect.Type) (reflect.Value, error) {
			str, ok := fromValue.Interface().(string)
			if ok {
				t, err := time.Parse(time.RFC3339, str)
				return reflect.ValueOf(t), err
			}
			return fromValue, nil
		},
	}
)

type Copier struct {
	from            interface{}
	converters      []Converter
	diffPairs       map[string][]string
	transformers    map[string]interface{}
	ignoreTypeError bool
}

type ConverterFunc = func(fromValue reflect.Value, toType reflect.Type) (toValue reflect.Value, err error)

type Converter struct {
	Origin reflect.Type
	Target reflect.Type
	Fn     ConverterFunc
}

type DiffPair struct {
	Origin string
	Target []string
}

func New(ignoreTypeError bool) *Copier {
	c := new(Copier)
	c.transformers = make(map[string]interface{})
	c.diffPairs = make(map[string][]string)
	c.ignoreTypeError = true
	return c
}

func (c *Copier) RegisterDiffPairs(pairs []DiffPair) *Copier {
	for _, pair := range pairs {
		c.diffPairs[pair.Origin] = pair.Target
	}
	return c
}

func (c *Copier) RegisterTransformer(field string, transformer interface{}) *Copier {
	if transformer == nil {
		panic("transformer cannot be nil")
	}
	ft := reflect.TypeOf(transformer)
	if ft.Kind() != reflect.Func {
		panic("transformer must be a function")
	}
	if ft.NumIn() != 1 {
		panic("transformer must has 1 arg")
	}
	if ft.NumOut() != 1 {
		panic("transformer must has 1 return value")
	}
	c.transformers[field] = transformer
	return c
}

func (c *Copier) RegisterConverter(converter Converter) *Copier {
	if converter.Fn == nil {
		panic("converter func cannot be nil")
	}
	c.converters = append(c.converters, converter)
	return c
}

func (c *Copier) getConverter(origin, target reflect.Type) *Converter {
	for _, converter := range c.converters {
		if converter.Origin == origin && converter.Target == target {
			return &converter
		}
	}
	return nil
}

func (c *Copier) From(from interface{}) *Copier {
	c.from = from
	return c
}

func (c *Copier) To(target interface{}) error {
	to := reflect.ValueOf(target)
	if to.Kind() != reflect.Ptr {
		return errors.New("copier target should be a pointer")
	}
	fromValue := reflect.ValueOf(c.from)
	if fromValue.Kind() == reflect.Ptr && (fromValue.IsNil() || !fromValue.IsValid()) {
		to.Set(reflect.Zero(to.Type()))
		return nil
	}
	return c.copyValue(fromValue, to, to.Type())
}

func (c *Copier) copyValue(from reflect.Value, to reflect.Value, toType reflect.Type) error {
	v, err := c.getTargetValue(getRealValue(from), getRealValue(to), getRealType(toType))
	if err != nil {
		return err
	}
	getRealValue(to).Set(v)
	return nil
}

func (c *Copier) getTargetValue(from reflect.Value, to reflect.Value, toType reflect.Type) (reflect.Value, error) {
	converter := c.getConverter(from.Type(), toType)
	if converter != nil {
		return converter.Fn(from, toType)
	} else if from.Type().ConvertibleTo(toType) {
		return from.Convert(toType), nil
	} else if from.Kind() == reflect.Ptr || to.Kind() == reflect.Ptr {
		if from.Kind() == reflect.Ptr && to.Kind() == reflect.Ptr {
			return c.getTargetValue(from.Elem(), to.Elem(), to.Elem().Type())
		} else if from.Kind() == reflect.Ptr {
			return c.getTargetValue(from.Elem(), to, toType)
		}
		return c.getTargetValue(from, to.Elem(), to.Elem().Type())
	} else if toType.Kind() == reflect.Struct && from.Kind() == reflect.Struct {
		return c.getStructTargetValue(from, to, toType)
	} else if from.Kind() == reflect.Slice && toType.Kind() == reflect.Slice {
		return c.getSliceTargetValue(from, toType)
	}
	err := errors.New(fmt.Sprintf("cannot convert value from %s to %s", from.Type().Name(), toType.Name()))
	if c.ignoreTypeError {
		err = nil
	}
	return reflect.Zero(toType), err
}

func (c *Copier) getStructTargetValue(from reflect.Value, to reflect.Value, toType reflect.Type) (reflect.Value, error) {
	if !to.IsValid() {
		to = reflect.New(toType).Elem()
	}
	toFieldsMap := getFieldMap(getStructAllFields(to.Type()))
	for _, fromField := range getStructAllFields(from.Type()) {
		fromValue := from.FieldByName(fromField.Name)
		if fromValue.IsValid() {
			targetFieldNames := c.getTargetFieldNames(fromField.Name)
			for _, targetFieldName := range targetFieldNames {
				toField, ok := toFieldsMap[targetFieldName]
				if ok {
					toValue := to.FieldByName(toField.Name)
					if toValue.IsValid() {
						sourceValue := fromValue
						if transformer, found := c.transformers[toField.Name]; found {
							fn := reflect.ValueOf(transformer)
							args := []reflect.Value{fromValue}
							returnValues := fn.Call(args)
							sourceValue = returnValues[0]
						}
						err := c.copyValue(sourceValue, toValue, toValue.Type())
						if err != nil {
							return to, err
						}
					}
				}
			}
		}
	}
	err := c.setMultiLevelFields(from, to, toType)
	return to, err
}

func (c *Copier) setMultiLevelFields(from reflect.Value, to reflect.Value, toType reflect.Type) error {
	for origin, targets := range c.diffPairs {
		for _, target := range targets {
			if strings.ContainsAny(origin, ".") || strings.ContainsAny(target, ".") {
				fromFieldValue := getValueByFieldPath(origin, from)
				toFieldValue := getValueByFieldPath(target, to)
				err := c.copyValue(fromFieldValue, toFieldValue, toFieldValue.Type())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *Copier) getSliceTargetValue(from reflect.Value, toType reflect.Type) (reflect.Value, error) {
	fromLength := from.Len()
	to := reflect.MakeSlice(toType, 0, fromLength)
	for i := 0; i < fromLength; i++ {
		originValue := from.Index(i)
		targetValue, err := c.getTargetValue(originValue, reflect.ValueOf(nil), getRealType(toType.Elem()))
		if err != nil {
			return to, err
		}
		if toType.Elem().Kind() == reflect.Ptr {
			to = reflect.Append(to, getValuePtr(targetValue))
		} else {
			to = reflect.Append(to, targetValue)
		}
	}
	return to, nil
}

func (c *Copier) getTargetFieldNames(origin string) []string {
	if len(c.diffPairs[origin]) > 0 {
		return c.diffPairs[origin]
	}
	return []string{origin}
}

func getRealValue(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		return v.Elem()
	}
	return v
}

func getRealType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}

func getValuePtr(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		return v
	}
	if v.CanAddr() {
		return v.Addr()
	}
	ptr := reflect.New(v.Type())
	ptr.Elem().Set(v)
	return ptr
}

func getStructAllFields(t reflect.Type) []reflect.StructField {
	fields := make([]reflect.StructField, 0)
	realType := getRealType(t)
	if realType.Kind() == reflect.Struct {
		for i := 0; i < realType.NumField(); i++ {
			field := realType.Field(i)
			if field.Anonymous {
				fields = append(fields, getStructAllFields(field.Type)...)
			} else {
				fields = append(fields, field)
			}
		}
	}
	return fields
}

func getFieldMap(fields []reflect.StructField) map[string]reflect.StructField {
	m := make(map[string]reflect.StructField, len(fields))
	for _, field := range fields {
		m[field.Name] = field
	}
	return m
}

// getValueByFieldPath this method extract value from struct by multi field path
// e.g. getValueByFieldPath("Product.Id", exampleStruct{})
func getValueByFieldPath(p string, value reflect.Value) reflect.Value {
	if strings.ContainsAny(p, ".") {
		paths := strings.Split(p, ".")
		first := paths[0]
		others := strings.Join(paths[1:], ".")
		return getValueByFieldPath(others, getValueByFieldPath(first, value))
	} else {
		if value.Kind() == reflect.Ptr {
			if value.IsNil() {
				return reflect.Zero(value.Type())
			}
			return value.Elem().FieldByName(p)
		} else if value.Kind() == reflect.Struct {
			return value.FieldByName(p)
		} else {
			panic(fmt.Sprintf("cannot get value by fields for: %s", value.Type().Name()))
		}
	}
}
