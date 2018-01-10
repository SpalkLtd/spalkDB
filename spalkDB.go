//author: Dion Bramley Jan 2018
package spalkDB

import (
	"bytes"
	"database/sql"
	"errors"
	"reflect"
	"unicode"

	"github.com/gocraft/dbr"
)

type InsertBuilder struct {
	*dbr.InsertBuilder
}

func (b *InsertBuilder) Set(column string, value interface{}) *InsertBuilder {
	dbrB := b.Pair(column, value)
	return &InsertBuilder{dbrB}
}

type Builder interface {
	Set(string, interface{}) Builder
	Exec() (sql.Result, error)
}

// the first parameter must satisfy Builder, or be a dbr.InsertBuilder
// The last parameter must be a struct
func MapStruct(b interface{}, cols []string, value interface{}) (*Builder, error) {
	switch v := b.(type) {
	case Builder:
		return mapStruct(v, cols, value)
	case *dbr.InsertBuilder:
		return mapStruct(&InsertBuilder{v}, cols, value)
	default:
		return nil, errors.New("First parameter to MapStruct must be a Builder, dbr.UpdateBuilder, or dbr.InsertBuilder")
	}
}

func mapStruct(b Builder, cols []string, value interface{}) (*Builder, error) {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Struct {
		return nil, errors.New("unsupported type passed in as value. value must be a struct.")
	}
	count := rv.NumField()
	fields := make([]reflect.Value, count)
	for i := 0; i < count; i++ {
		fields[i] = rv.FieldByIndex(i)
	}
	if cols == nil {
		//populate from value using reflection
		cols = make([]string, 0)
		for _, f := range fields {
			if unicode.IsLower(f.Name[0]) && f.Tag.Get("db") != "-" && (f.Kind <= reflect.Complex128 || f.Kind == reflect.String) {
				tag := f.Tag.Get("db")
				if tag != "" {
					cols = append(cols, tag)
				}
				cols = append(cols, camelCaseToSnakeCase(f.Name.String()))
			}
		}
	}

colLoop:
	for c := range cols {
		matches := matchName(c)
		f := rv.FieldByNameFunc(matches)
		if f.IsValid() {
			b.Set(c, f.Interface())
		} else {
			for v := range feilds {
				if f.Tag.Get("db") == c || matches(f.Name) {
					b.Set(c, f.Interface())
					continue colLoop
				}
			}
			return errors.New("no match found for column %s in struct", c)
		}
	}

	return b, nil
}

func matchName(col string) func(string) bool {
	return func(string name) bool {
		return col == name || col == camelCaseToSnakeCase(name)
	}
}

//coppied from dbr/util.go for compatibility
func camelCaseToSnakeCase(name string) string {
	buf := new(bytes.Buffer)

	runes := []rune(name)

	for i := 0; i < len(runes); i++ {
		buf.WriteRune(unicode.ToLower(runes[i]))
		if i != len(runes)-1 && unicode.IsUpper(runes[i+1]) &&
			(unicode.IsLower(runes[i]) || unicode.IsDigit(runes[i]) ||
				(i != len(runes)-2 && unicode.IsLower(runes[i+2]))) {
			buf.WriteRune('_')
		}
	}

	return buf.String()
}
