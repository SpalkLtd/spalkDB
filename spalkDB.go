//author: Dion Bramley Jan 2018
package spalkDB

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"unicode"

	"github.com/gocraft/dbr"
)

// the first parameter must satisfy Builder, or be a dbr.InsertBuilder
// The last parameter must be a struct
func MapStruct(b interface{}, cols []string, value interface{}) func() (sql.Result, error) {
	switch b.(type) {
	case *dbr.UpdateBuilder:
	case *dbr.InsertBuilder:
	default:
		panic(errors.New("First parameter to MapStruct must be a Builder, dbr.UpdateBuilder, or dbr.InsertBuilder"))
	}

	rt := reflect.TypeOf(value)
	if rt.Kind() != reflect.Struct {
		panic(errors.New("unsupported type passed in as value. value must be a struct."))
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
			if unicode.IsLower(rune(f.Name[0])) && f.Tag.Get("db") != "-" && (f.Type.Kind() <= reflect.Complex128 || f.Type.Kind() == reflect.String) {
				tag := f.Tag.Get("db")
				if tag != "" {
					cols = append(cols, tag)
				}
				cols = append(cols, camelCaseToSnakeCase(f.Name))
			}
		}
	}

	rf := reflect.ValueOf(value)

colLoop:
	for _, c := range cols {
		matches := matchName(c)
		// data, ok := rf.FieldByIndex(f.Index)
		data := rf.FieldByNameFunc(matches)
		if data.IsValid() {
			switch v := b.(type) {
			case *dbr.UpdateBuilder:
				v.Set(c, data.Interface())
			case *dbr.InsertBuilder:
				v.Pair(c, data.Interface())
			}

		} else {
			for _, f := range fields {
				if f.Tag.Get("db") == c || matches(f.Name) {
					data := rf.FieldByIndex(f.Index)
					// b.Set(c, f.Interface())
					switch v := b.(type) {
					case *dbr.UpdateBuilder:
						v.Set(c, data.Interface())
					case *dbr.InsertBuilder:
						v.Pair(c, data.Interface())
					}
					continue colLoop
				}
			}
			panic(errors.New(fmt.Sprintf("no match found for column %s in struct", c)))
		}
	}

	switch v := b.(type) {
	case *dbr.UpdateBuilder:
		return v.Exec
	case *dbr.InsertBuilder:
		return v.Exec
	}
	return nil
}

func matchName(col string) func(string) bool {
	return func(name string) bool {
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
