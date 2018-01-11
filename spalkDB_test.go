package spalkDB

import (
	"fmt"
	"log"
	"reflect"
	"testing"

	"github.com/gocraft/dbr"
	"github.com/gocraft/dbr/dialect"
)

type matchTest struct {
	name  string
	col   string
	match bool
}

var matchList = []matchTest{
	matchTest{"A", "A", true},
	matchTest{"A", "b", false},
	matchTest{"A", "a", true},
	matchTest{"PascalCaseName", "pascal_case_name", true},
	matchTest{"PascalCaseName", "pascal_caseName", false},
	matchTest{"Pascal_caseName", "pascal_caseName", false},
	matchTest{"PascalCaseName", "pascal_case_Name", false},
	matchTest{"Pascal_case_Name", "Pascal_case_Name", true},
	matchTest{"snake_case_name", "snake_case_name", false}, //unexported names not allowed
	matchTest{"Snake_case_name", "snake_case_name", true},
}

func TestMatchName(t *testing.T) {
	for _, m := range matchList {
		if matchName(m.col)(m.name) != m.match {
			var not string
			if !m.match {
				not = " not"
			}
			fmt.Printf("%s should%s match %s\n", m.col, not, m.name)
			t.Fail()
		}
	}
}

type mapTest struct {
	value  interface{}
	cols   []string
	insert string
	update map[string]interface{}
	panics bool
}

func TestMapStruct(t *testing.T) {
	var q string
	var u *dbr.UpdateStmt
	var i *dbr.InsertStmt
	for _, m := range mapTestList {
		if m.panics {
			assertPanic(func() {
				u = dbr.Update("tableName")
				MapStruct(&dbr.UpdateBuilder{UpdateStmt: u}, m.cols, m.value)
			}, t)
		} else {
			u = dbr.Update("tableName")
			MapStruct(&dbr.UpdateBuilder{UpdateStmt: u}, m.cols, m.value)
			// q = getQueryString(u)
			// if q != m.update {
			// 	fmt.Printf("expected %+v but got %+v\n", m.update, q)
			// 	t.Fail()
			// }
			if !reflect.DeepEqual(m.update, u.Value) {
				fmt.Printf("expected %+v but got %+v\n", m.update, u.Value)
				t.Fail()
			}

			i = dbr.InsertInto("tableName")
			MapStruct(&dbr.InsertBuilder{InsertStmt: i}, m.cols, m.value)
			q = getQueryString(i)
			if q != m.insert {
				fmt.Printf("expected %+v but got %+v\n", m.insert, q)
				t.Fail()
			}
		}
	}
}

var mapTestList = []mapTest{
	mapTest{ // correct mapping of single value
		struct{ Name string }{"name"},
		nil,
		"INSERT INTO `tableName` (`name`) VALUES ('name')",
		map[string]interface{}{"name": "name"},
		false,
	},
	mapTest{ // doesn't try to use unexported values
		struct{ name string }{"name"},
		nil,
		"",
		map[string]interface{}{},
		false,
	},
	mapTest{ // Correctly handle multible values and different data types(should be handled by dbr)
		struct {
			Foo  string
			Name string
			Bar  int
		}{"Foo", "name", 7},
		nil,
		"INSERT INTO `tableName` (`foo`,`name`,`bar`) VALUES ('Foo','name',7)",
		map[string]interface{}{"name": "name", "foo": "Foo", "bar": 7},
		false,
	},
	mapTest{ // combination of public and private attribs
		struct {
			Foo     string
			Name    string
			missing string
		}{"id", "name", "missing"},
		nil,
		"INSERT INTO `tableName` (`foo`,`name`) VALUES ('id','name')",
		map[string]interface{}{"name": "name", "foo": "id"},
		false,
	},
	mapTest{ // tag same as field name, tag to omit, and omit private field despite tag to include
		struct {
			Foo     string `db:"Foo"`
			Name    string `db:"-"`
			missing string `db:"missing"`
		}{"id", "name", "missing"},
		nil,
		"INSERT INTO `tableName` (`Foo`) VALUES ('id')",
		map[string]interface{}{"Foo": "id"},
		false,
	},
	mapTest{ // panic on non-existant columns
		struct {
			missing string `db:"missing"`
		}{"missing"},
		[]string{"id", "name", "missing"},
		"",
		nil,
		true,
	},
	mapTest{ // panic on not matching specified column because col is name not tag
		struct {
			Something string `db:"notAnId"`
		}{"id"},
		[]string{"something"},
		"",
		nil,
		true,
	},
	mapTest{ // don't match col to field if omitted by tag
		struct {
			Name string `db:"-"`
		}{"name"},
		[]string{"name"},
		"",
		nil,
		true,
	},
	mapTest{ // correctly handle tag being different to field
		struct {
			Foo     string `db:"notAnId"`
			Name    string `db:"-"`
			missing string
		}{"id", "name", "missing"},
		nil,
		"INSERT INTO `tableName` (`notAnId`) VALUES ('id')",
		map[string]interface{}{"notAnId": "id"},
		false,
	},
	mapTest{ // Match multiple fields in order in cols list
		struct {
			Foo     string
			Name    string
			missing string `db:"missing"`
		}{"id", "name", "missing"},
		[]string{"foo", "name"},
		"INSERT INTO `tableName` (`foo`,`name`) VALUES ('id','name')",
		map[string]interface{}{"foo": "id", "name": "name"},
		false,
	},
	mapTest{ // reverse order from above to be sure
		struct {
			Foo     string
			Name    string
			missing string `db:"missing"`
		}{"id", "name", "missing"},
		[]string{"name", "foo"},
		"INSERT INTO `tableName` (`name`,`foo`) VALUES ('name','id')",
		map[string]interface{}{"foo": "id", "name": "name"},
		false,
	},
	mapTest{ // omit id for update but not insert. Match col to renamed field
		struct {
			ID      string
			Name    string `db:"blarg"`
			Omitted string
		}{"id", "name", "missing"},
		[]string{"blarg", "id"},
		"INSERT INTO `tableName` (`blarg`,`id`) VALUES ('name','id')",
		map[string]interface{}{"blarg": "name"},
		false,
	},

	mapTest{ // correctly omit id
		struct {
			ID   string
			Name string
		}{"id", "name"},
		nil,
		"INSERT INTO `tableName` (`id`,`name`) VALUES ('id','name')",
		map[string]interface{}{"name": "name"},
		false,
	},
}

//should return "" if failed
func getQueryString(b dbr.Builder) string {
	buf := dbr.NewBuffer()
	err := b.Build(dialect.MySQL, buf)
	if err != nil {
		log.Println(err.Error())
		return ""
	}

	query := buf.String()

	query, err = dbr.InterpolateForDialect(query, buf.Value(), dialect.MySQL)
	// log.Println(query)
	if err != nil {
		// log.Println(err.Error())
		return ""
	}
	return (query)
}

func assertPanic(f func(), t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	f()
}
