package copier

import (
	"errors"
	"reflect"
)

type Copier struct {
	from reflect.Value
}

func (c *Copier) From(s interface{}) *Copier {
	c.from = reflect.ValueOf(s)

	return c
}

func (c *Copier) To(target interface{}) error {
	to := reflect.ValueOf(target)
	if to.Kind() != reflect.Ptr {
		return errors.New("copier target should be a pointer")
	}
	if c.from.Kind() == reflect.Ptr && c.from.IsNil() || !c.from.IsValid() {
		to.Set(reflect.Zero(to.Type()))
		return nil
	}
	return copyValue(c.from, to, to.Type())
}

func copyValue(from reflect.Value, to reflect.Value, toType reflect.Type) error {

	return nil
}

func copyStruct(from, to reflect.Value, toType reflect.Type) error {
	return nil
}

func copySlice(from, to reflect.Value, toType reflect.Type) error {
	return nil
}
