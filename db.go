package main

import (
  "fmt"
  "database/sql"
  _ "github.com/ncruces/go-sqlite3/driver"
  _ "github.com/ncruces/go-sqlite3/embed"
  _ "embed"
  "gotenburg/template"
)

//go:embed sql/schema.sql
var schema string

func DbConnect() {
  db, _ := sql.Open("sqlite3", "file:app.db")
  if _, err := db.Exec(schema); err != nil {
    fmt.Println("Couldn't set up database", err)
    return
  }
  r := template.TemplateRepository{Db:db}

  t, err := r.Get(1)

  if err != nil {
    fmt.Println("Fail 3", err)
  } else {
    if t != nil {
      fmt.Println(t)
    } else {
      fmt.Println("empty")
    }
  }
}
