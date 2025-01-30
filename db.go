package main

import (
	"database/sql"
	_ "embed"
	"fmt"
	"image/png"
	"os"

	"tomgalvin.uk/phogoprint/template"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

//go:embed resources/sql/schema.sql
var schema string

func DbConnect(t0 *template.Template) {
	db, _ := sql.Open("sqlite3", "file:app.db")
	if _, err := db.Exec(schema); err != nil {
		fmt.Println("Couldn't set up database", err)
		return
	}
	r := template.TemplateRepository{Db: db}

	err := r.Transact(func(tx *sql.Tx) error {
		return r.Create(tx, t0)
	})
	if err != nil {
		fmt.Println("Error inserting", err)
		return
	}

	t, err := r.Get(t0.Id)

	if err != nil {
		fmt.Println("Fail 3", err)
	} else {
		if t != nil {
		} else {
			fmt.Println("empty")
		}
	}

	rendered, err := template.RenderTemplate(t, map[string]string{
		"param1": "bob saget bob saget hello!",
		"param2": "world! aeuio aeuoio aeuoi aeou aeouaeo ui",
	})
	if err != nil {
		panic(err)
	}

	// rfd := bitmap.RenderForDevice(rendered)

	outFile, err := os.Create("output2.png")
	if err != nil {
		panic(err)
	}
	defer outFile.Close()
	png.Encode(outFile, rendered)
}
