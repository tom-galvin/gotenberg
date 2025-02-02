package template

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type TemplateRepository struct {
  Db *sql.DB
}

func (r *TemplateRepository) Close() error {
	return r.Db.Close()
}

func (r *TemplateRepository) readTemplateBase(u uuid.UUID) (*Template, error) {
  row := r.Db.QueryRow(`
    SELECT id, name, created_at, landscape, min_size, max_size
    FROM template
    WHERE uuid = ?`, u.String())

	t := Template{Uuid: u}
  if err := row.Scan(&t.Id, &t.Name, &t.CreatedAt, &t.Landscape, &t.MinSize, &t.MaxSize); err != nil {
    if errors.Is(err, sql.ErrNoRows) {
      return nil, nil
    } else {
      return nil, fmt.Errorf("Failed to read template:\n%w", err)
    }
  }

  return &t, nil
}

func (r *TemplateRepository) ListFonts() ([]Font, error) {
	rows, err := r.Db.Query(`
	  SELECT uuid, name, builtin_name, font_data
		FROM font`)

	if err != nil {
		return nil, fmt.Errorf("Query execution failed:\n%w", err)
	}
	defer rows.Close()

	fonts := []Font{}
  for count := 0; rows.Next(); count++ {
		f := Font{}
		var uuidString string
		if err := rows.Scan(&uuidString, &f.Name, &f.BuiltinName, &f.FontData); err != nil {
			return nil, fmt.Errorf("Row scanning failed:\n%w", err)
		}
		f.Uuid = uuid.MustParse(uuidString)
		fonts = append(fonts, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error iterating rows:\n%w", err)
	}

	return fonts, nil
}

func (r *TemplateRepository) GetFont(u uuid.UUID) (*Font, error) {
	row := r.Db.QueryRow(`
	  SELECT uuid, name, builtin_name, font_data
		FROM font
		WHERE uuid = ?`, u.String())

	var f Font

	var uuidString string
	if err := row.Scan(&uuidString, &f.Name, &f.BuiltinName, &f.FontData); err != nil {
    if errors.Is(err, sql.ErrNoRows) {
      return nil, nil
    } else {
      return nil, fmt.Errorf("Failed to read font:\n%w", err)
    }
	}
	f.Uuid = uuid.MustParse(uuidString)

	return &f, nil
}

func (r *TemplateRepository) List() ([]Template, error) {
	rows, err := r.Db.Query(`SELECT uuid, id, name, landscape, min_size, max_size FROM template`)
	if err != nil {
		return nil, fmt.Errorf("Query execution failed:\n%w", err)
	}
	defer rows.Close()

	templates := []Template{}
  for count := 0; rows.Next(); count++ {
		t := Template{}
		var uuidString string
		if err := rows.Scan(&uuidString, &t.Id, &t.Name, &t.Landscape, &t.MinSize, &t.MaxSize); err != nil {
			return nil, fmt.Errorf("row scanning failed:\n%w", err)
		}
		t.Uuid = uuid.MustParse(uuidString)
		templates = append(templates, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error iterating rows:\n%w", err)
	}

	return templates, nil
}

func (r *TemplateRepository) Exists(u uuid.UUID) (bool, error) {
  t, err := r.readTemplateBase(u)
  if err != nil {
    return false, err
  }
	return (t != nil), nil
}

func (r *TemplateRepository) Get(u uuid.UUID) (*Template, error) {
  t, err := r.readTemplateBase(u)
  if err != nil {
    return nil, err
  }
  if t == nil {
    return nil, nil
  }

  var paramCount, imageCount, textCount int
  row := r.Db.QueryRow(`
    SELECT
      (SELECT COUNT(1) FROM template_parameter WHERE template_id = ?) AS param_count,
      (SELECT COUNT(1) FROM template_image WHERE template_id = ?) AS image_count,
      (SELECT COUNT(1) FROM template_text WHERE template_id = ?) AS text_count
    `, t.Id, t.Id, t.Id)

  if err := row.Scan(&paramCount, &imageCount, &textCount); err != nil {
    return nil, fmt.Errorf("Failed to query template child count:\n%w", err)
  }

  t.Parameters = make([]Parameter, paramCount)
  if err := QueryAndScanRows(r.Db, `
    SELECT id, name, max_length
    FROM template_parameter
    WHERE template_id = ?`, t.Id, t.Parameters, func(r *sql.Rows, x *Parameter) error {
      return r.Scan(&x.Id, &x.Name, &x.MaxLength)
    },
  ); err != nil {
    return nil, fmt.Errorf("Failed to read parameters for template:\n%w", err)
  }

  t.Images = make([]Image, imageCount)
  if err := QueryAndScanRows(r.Db, `
    SELECT id, image, x, y, width, height
    FROM template_image
    WHERE template_id = ?`, t.Id, t.Images, func(r *sql.Rows, i *Image) error {
      return r.Scan(&i.Id, &i.Image, &i.X, &i.Y, &i.Width, &i.Height)
    },
  ); err != nil {
    return nil, fmt.Errorf("Failed to read child images for template:\n%w", err)
  }

  t.Texts = make([]Text, textCount)
  if err := QueryAndScanRows(r.Db, `
    SELECT t.id, t.text, t.x, t.y, t.width, t.height, t.font_size, f.uuid, f.name, f.builtin_name, f.font_data
    FROM template_text t
		JOIN font f ON f.id = t.font_id
    WHERE t.template_id = ?`, t.Id, t.Texts, func(r *sql.Rows, i *Text) error {
			var uuidString string
			err := r.Scan(
				&i.Id, &i.Text, &i.X, &i.Y, &i.Width, &i.Height, &i.FontSize,
				&uuidString, &i.Font.Name, &i.Font.BuiltinName, &i.Font.FontData)
			i.Font.Uuid = uuid.MustParse(uuidString)
			return err
    },
  ); err != nil {
    return nil, fmt.Errorf("Failed to read child texts for image:\n%w", err)
  }

  return t, nil
}

func QueryAndScanRows[T any](db *sql.DB, query string, id int, results []T, scanRow func(*sql.Rows, *T) error) error {
	rows, err := db.Query(query, id)
	if err != nil {
		return fmt.Errorf("Query execution failed:\n%w", err)
	}
	defer rows.Close()

  for count := 0; rows.Next(); count++ {
		if count >= len(results) {
      panic("preallocated slice size is smaller than the number of rows returned") // shouldn't happen
		}

		if err := scanRow(rows, &results[count]); err != nil {
			return fmt.Errorf("row scanning failed:\n%w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("Error iterating rows:\n%w", err)
	}

	return nil
}

// Run operations in a transaction, committing afterward, or rolling back if the
// passed function returns an error
func (r *TemplateRepository) Transact(f func(*sql.Tx) error) error {
  tx, err := r.Db.Begin()
  if err != nil {
    return err
  }

  err = f(tx)
  if err != nil {
    err2 := tx.Rollback()
    if err2 != nil {
      return fmt.Errorf("Failed to roll back transaction: %w\n\nAfter handling: %v", err2, err)
    }
    return err
  } else {
    err2 := tx.Commit()
    if err2 != nil {
      return fmt.Errorf("Failed to commit transaction:\n%w", err2)
    }
    return nil
  }
}

func (r *TemplateRepository) Multi(tx *sql.Tx, param any, qs ...string) error {
	for n, q := range qs {
		if _, err := tx.Exec(q, param); err != nil {
			return fmt.Errorf("Error running statement #%d:\n%w", n+1, err)
		}
	}
	return nil
}

func (r *TemplateRepository) Create(tx *sql.Tx, t *Template) error {
  row := tx.QueryRow(`
    INSERT INTO template(uuid, name, created_at, landscape, min_size, max_size)
    VALUES (?, ?, ?, ?, ?, ?)
    RETURNING id`, t.Uuid.String(), t.Name, t.CreatedAt, t.Landscape, t.MinSize, t.MaxSize)
  if err := row.Scan(&t.Id); err != nil {
    return fmt.Errorf("Failed to insert into template:\n%w", err)
  }

  r.insertChildren(tx, t)

  return nil
}

func (r *TemplateRepository) Update(tx *sql.Tx, u uuid.UUID, t *Template) error {
  tFromDb, err := r.readTemplateBase(t.Uuid)
  if err != nil {
    return err
  }
  if tFromDb == nil {
    return fmt.Errorf("No template with UUID %s", u.String())
  }
	
	t.Id = tFromDb.Id
	if err := r.Multi(tx, t.Id,
		  "DELETE FROM template_parameter WHERE template_id = ?",
      "DELETE FROM template_image WHERE template_id = ?",
      "DELETE FROM template_text WHERE template_id = ?"); err != nil {
		return err
  }

  _, err = tx.Exec(`UPDATE template SET name = ?, landscape = ? WHERE id = ?`,
    t.Name, t.Landscape, t.Id)
  if err != nil {
    return fmt.Errorf("Couldn't update template data:\n%w", err)
  }
  if err := r.insertChildren(tx, t); err != nil {
    return err
  }

  return nil
}

func (r *TemplateRepository) insertChildren(tx *sql.Tx, t *Template) error {
  pStmt, err := tx.Prepare(`
    INSERT INTO template_parameter(template_id, name, max_length)
    VALUES (?, ?, ?)`)
  if err != nil {
    return fmt.Errorf("Failed to prepare statement to insert template parameter:\n%w", err)
  }
  defer pStmt.Close()
  for i, p := range t.Parameters {
    _, err := pStmt.Exec(t.Id, p.Name, p.MaxLength)
    if err != nil {
      return fmt.Errorf("Failed to insert parameter %v of template:\n%w", i, err)
    }
  }

  iStmt, err := tx.Prepare(`
    INSERT INTO template_image(template_id, image, x, y, width, height)
    VALUES (?, ?, ?, ?, ?, ?)`)
  if err != nil {
    return fmt.Errorf("Failed to prepare statement to insert template image:\n%w", err)
  }
  defer iStmt.Close()
  for i, img := range t.Images {
    _, err := iStmt.Exec(t.Id,
      img.Image,
      img.X, img.Y,
      img.Width,
      img.Height,
    )
    if err != nil {
      return fmt.Errorf("Failed to insert parameter %v of image:\n%w", i, err)
    }
  }

  tStmt, err := tx.Prepare(`
    INSERT INTO template_text(template_id, text, x, y, width, height, font_size, font_id)
    VALUES (?, ?, ?, ?, ?, ?, ?, (SELECT id FROM font WHERE uuid = ?))`)
  if err != nil {
    return fmt.Errorf("Failed to prepare statement to insert template text:\n%w", err)
  }
  defer tStmt.Close()
  for i, txt := range t.Texts {
    _, err := tStmt.Exec(t.Id,
      txt.Text,
      txt.X, txt.Y,
      txt.Width,
      txt.Height,
			txt.FontSize,
			txt.Font.Uuid.String(),
    )
    if err != nil {
      return fmt.Errorf("Failed to insert parameter %v of text:\n%w", i, err)
    }
  }

  return nil
}
