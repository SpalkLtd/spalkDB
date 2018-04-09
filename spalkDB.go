// Package spalkDB provides a utility for easily inserting structs into a database
package spalkDB

//author: Dion Bramley Jan 2018

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/SpalkLtd/dbr"
)

// MapStruct marshalls a struct into a database query, for use with gocraft/dbr
// the first parameter must satisfy Builder, or be a dbr.InsertBuilder
// The last parameter must be a struct
func MapStruct(b interface{}, cols []string, value interface{}) func() (sql.Result, error) {
	if reflect.TypeOf(value).Kind() == reflect.Ptr {
		value = reflect.ValueOf(value).Elem().Interface()
	}
	switch b.(type) {
	case *dbr.UpdateBuilder:
	case *dbr.InsertBuilder:
	default:
		panic(errors.New("First parameter to MapStruct must be a Builder, dbr.UpdateBuilder, or dbr.InsertBuilder but got " + reflect.TypeOf(b).String()))
	}

	rt := reflect.TypeOf(value)
	if rt.Kind() != reflect.Struct {
		panic(errors.New("unsupported type passed in as value. value must be a struct. Instead got: " + rt.String()))
	}
	count := rt.NumField()
	fields := make([]reflect.StructField, count)
	for i := 0; i < count; i++ {
		fields[i] = rt.FieldByIndex([]int{i})
	}
	if cols == nil {
		//populate from value using reflection
		cols = make([]string, 0)
		for _, f := range fields {
			if f.Tag.Get("db") != "-" && (f.Type.Kind() <= reflect.Complex128 || f.Type.Kind() == reflect.String) && unicode.IsUpper(rune(f.Name[0])) {
				tag := f.Tag.Get("db")
				if tag != "" {
					cols = append(cols, tag)
				} else {
					cols = append(cols, camelCaseToSnakeCase(f.Name))
				}
			}
		}
	}
	// log.Println(cols)

	rf := reflect.ValueOf(value)

colLoop:
	for _, col := range cols {
		matches := matchName(col)
		// data, ok := rf.FieldByIndex(f.Index)
		f, ok := rt.FieldByNameFunc(matches)
		if ok && f.Tag.Get("db") == "" {
			data := rf.FieldByNameFunc(matches)
			if data.IsValid() {
				set(b, col, data)
				continue
			}
		}

		for _, f := range fields {
			tag := f.Tag.Get("db")
			if unicode.IsUpper(rune(f.Name[0])) && tag != "-" && (tag == col || (tag == "" && matches(f.Name))) {
				data := rf.FieldByIndex(f.Index)
				// b.Set(c, f.Interface())
				set(b, col, data)
				continue colLoop
			}
		}
		panic(fmt.Errorf("no match found for column %s in struct %s", col, rt.String()))

	}

	switch v := b.(type) {
	case *dbr.UpdateBuilder:
		return v.Exec
	case *dbr.InsertBuilder:
		return v.Exec
	}
	return nil
}

func set(b interface{}, c string, data reflect.Value) {
	switch v := b.(type) {
	case *dbr.UpdateBuilder:
		if strings.ToLower(c) == "id" {
			v.Where("id=?", data.Interface())
		} else {
			v.Set(c, data.Interface())
		}
	case *dbr.InsertBuilder:
		v.Pair(c, data.Interface())
	}
}

func matchName(col string) func(string) bool {
	return func(name string) bool {
		return unicode.IsUpper(rune(name[0])) && (col == name || col == camelCaseToSnakeCase(name))
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
