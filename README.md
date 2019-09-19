# spalkDB
utility to make dbr more user friendly

[![GoDoc](https://godoc.org/github.com/SpalkLtd/spalkDB?status.svg)](https://godoc.org/github.com/SpalkLtd/spalkDB)
[![Go Report Card](https://goreportcard.com/badge/github.com/spalkLtd/spalkdb)](https://goreportcard.com/badge/github.com/spalkLtd/spalkdb)
[![Travis Build](https://travis-ci.com/SpalkLtd/spalkDB.svg?branch=master)](https://travis-ci.com/SpalkLtd/spalkDB.svg?branch=master)

## Usage
Parameters:
 - First parameter is either a *dbr.InsertBuilder or a *dbr.UpdateBuilder
 - Second parameter is an optional list of column names to map into the query
 - Third parameter is the struct that should be used to source data

Returns the Exec() function from the builder passed in

### Add every field in a struct to the query
```go
// from dbr docs: set up a dbr session
// create a connection (e.g. "postgres", "mysql", or "sqlite3")
conn, _ := dbr.Open("postgres", "...")

// create a session for each business unit of execution (e.g. a web request or goworkers job)
sess := conn.NewSession(nil)

// Then pass the query into MapStruct and immediately execute the query
_,err := MapStruct(sess.InsertInto("tableName"), nil, myData)()
```

