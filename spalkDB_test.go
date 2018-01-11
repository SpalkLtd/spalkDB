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
	mapTest{
		struct{ Name string }{"name"},
		nil,
		"INSERT INTO `tableName` (`name`) VALUES ('name')",
		map[string]interface{}{"name": "name"},
		false,
	},
	mapTest{
		struct{ name string }{"name"},
		nil,
		"",
		map[string]interface{}{},
		false,
	},
	mapTest{
		struct {
			ID   string
			Name string
		}{"id", "name"},
		nil,
		"INSERT INTO `tableName` (`id`,`name`) VALUES ('id','name')",
		map[string]interface{}{"name": "name", "id": "id"},
		false,
	},
	mapTest{
		struct {
			ID      string
			Name    string
			missing string
		}{"id", "name", "missing"},
		nil,
		"INSERT INTO `tableName` (`id`,`name`) VALUES ('id','name')",
		map[string]interface{}{"name": "name", "id": "id"},
		false,
	},
	mapTest{
		struct {
			ID      string `db:"ID"`
			Name    string `db:"-"`
			missing string `db:"missing"`
		}{"id", "name", "missing"},
		nil,
		"INSERT INTO `tableName` (`ID`) VALUES ('id')",
		map[string]interface{}{"ID": "id"},
		false,
	},
	mapTest{
		struct {
			missing string `db:"missing"`
		}{"missing"},
		[]string{"id", "name", "missing"},
		"",
		nil,
		true,
	},
	mapTest{
		struct {
			ID string `db:"notAnId"`
		}{"id"},
		[]string{"name", "id"},
		"",
		nil,
		true,
	},
	mapTest{
		struct {
			Name string `db:"-"`
		}{"name"},
		[]string{"name", "id"},
		"",
		nil,
		true,
	},
	mapTest{
		struct {
			ID      string `db:"notAnId"`
			Name    string `db:"-"`
			missing string
		}{"id", "name", "missing"},
		nil,
		"INSERT INTO `tableName` (`notAnId`) VALUES ('id')",
		map[string]interface{}{"notAnId": "id"},
		false,
	},
	mapTest{
		struct {
			ID      string
			Name    string
			missing string `db:"missing"`
		}{"id", "name", "missing"},
		[]string{"id", "name"},
		"INSERT INTO `tableName` (`id`,`name`) VALUES ('id','name')",
		map[string]interface{}{"id": "id", "name": "name"},
		false,
	},
	mapTest{
		struct {
			ID      string
			Name    string
			missing string `db:"missing"`
		}{"id", "name", "missing"},
		[]string{"name", "id"},
		"INSERT INTO `tableName` (`name`,`id`) VALUES ('name','id')",
		map[string]interface{}{"id": "id", "name": "name"},
		false,
	},
	mapTest{
		struct {
			ID      string
			Name    string `db:"blarg"`
			Omitted string
		}{"id", "name", "missing"},
		[]string{"blarg", "id"},
		"INSERT INTO `tableName` (`blarg`,`id`) VALUES ('name','id')",
		map[string]interface{}{"id": "id", "blarg": "name"},
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
	log.Println(query)
	if err != nil {
		log.Println(err.Error())
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
