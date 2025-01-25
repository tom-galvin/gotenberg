package main

import (
  "fmt"
  "database/sql"
  _ "github.com/ncruces/go-sqlite3/driver"
  _ "github.com/ncruces/go-sqlite3/embed"
  _ "embed"
)

//go:embed sql/schema.sql
var schema string

func DbConnect() {
  var rows int
  db, _ := sql.Open("sqlite3", "file:app.db")
  fmt.Println(schema)
  if _, err := db.Exec(schema); err != nil {
    fmt.Println("Fail 1", err)
    return
  }
  
  if err := db.QueryRow(`SELECT COUNT(*) FROM print_queue`).Scan(&rows); err != nil {
    fmt.Println( "Fail 2", err)
    return
  }
  fmt.Printf("From DB: %d\n", rows)
}
