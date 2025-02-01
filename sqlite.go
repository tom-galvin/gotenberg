package main

import (
	"database/sql"
	_ "embed"
	"fmt"

	"tomgalvin.uk/phogoprint/internal/template"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

//go:embed resources/sql/schema.sql
var schema string

func NewRepository() *template.TemplateRepository {
	var db *sql.DB
	var err error

	if db, err = sql.Open("sqlite3", "file:app.db"); err != nil {
		panic(fmt.Errorf("Couldn't open database:\n%w", err))
	}
	if _, err := db.Exec(schema); err != nil {
		panic(fmt.Errorf("Couldn't initialise database:\n%w", err))
	}
	r := template.TemplateRepository{Db: db}

	return &r
}
