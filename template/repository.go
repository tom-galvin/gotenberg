package template

import (
  "database/sql"
  "errors"
  "fmt"
)

type TemplateRepository struct {
  Db *sql.DB
}

func (r *TemplateRepository) readTemplateBase(id int) (*Template, error) {
  row := r.Db.QueryRow(`
    SELECT name, created_at, landscape
    FROM template
    WHERE id = ?`, id)

  t := Template{Id: id}
  if err := row.Scan(&t.Name, &t.CreatedAt, &t.Landscape); err != nil {
    if errors.Is(err, sql.ErrNoRows) {
      return nil, nil
    } else {
      return nil, fmt.Errorf("Failed to read template:\n%w", err)
    }
  }

  return &t, nil
}

func (r *TemplateRepository) Get(id int) (*Template, error) {
  t, err := r.readTemplateBase(id)
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
    `, id, id, id)

  if err := row.Scan(&paramCount, &imageCount, &textCount); err != nil {
    return nil, fmt.Errorf("Failed to query template child count:\n%w", err)
  }

  t.Parameters = make([]Parameter, paramCount)
  if err := QueryAndScanRows(r.Db, `
    SELECT id, name, max_length
    FROM template_parameter
    WHERE template_id = ?`, id, t.Parameters, func(r *sql.Rows, x *Parameter) error {
      return r.Scan(&x.Id, &x.Name, &x.MaxLength)
    },
  ); err != nil {
    return nil, fmt.Errorf("Failed to read parameters for template:\n%w", err)
  }

  t.Images = make([]Image, imageCount)
  if err := QueryAndScanRows(r.Db, `
    SELECT id, image, x, y, width, height
    FROM template_image
    WHERE template_id = ?`, id, t.Images, func(r *sql.Rows, i *Image) error {
      return r.Scan(&i.Id, &i.Image, &i.X, &i.Y, &i.Width, &i.Height)
    },
  ); err != nil {
    return nil, fmt.Errorf("Failed to read child images for template:\n%w", err)
  }

  t.Texts = make([]Text, textCount)
  if err := QueryAndScanRows(r.Db, `
    SELECT id, text, x, y, width, height
    FROM template_text
    WHERE template_id = ?`, id, t.Texts, func(r *sql.Rows, i *Text) error {
      return r.Scan(&i.Id, &i.Text, &i.X, &i.Y, &i.Width, &i.Height)
    },
  ); err != nil {
    return nil, fmt.Errorf("Failed to read child texts for image::\n%w", err)
  }

  return t, nil
}

func QueryAndScanRows[T any](db *sql.DB, query string, id int, results []T, scanRow func(*sql.Rows, *T) error) error {
	rows, err := db.Query(query, id)
	if err != nil {
		return fmt.Errorf("Query execution failed:\n%w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		if count >= len(results) {
      // shouldn't happen!
      panic("preallocated slice size is smaller than the number of rows returned")
		}

		// Scan the row into the current element
		if err := scanRow(rows, &results[count]); err != nil {
			return fmt.Errorf("row scanning failed:\n%w", err)
		}
		count++
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

func (r *TemplateRepository) Create(tx *sql.Tx, t *Template) error {
  row := tx.QueryRow(`
    INSERT INTO template(name, created_at, landscape)
    VALUES (?, ?, ?)
    RETURNING id`, t.Name, t.CreatedAt, t.Landscape)
  if err := row.Scan(&t.Id); err != nil {
    return fmt.Errorf("Failed to insert into template:\n%w", err)
  }

  r.insertChildren(tx, t)

  return nil
}

func (r *TemplateRepository) Update(tx *sql.Tx, id int, t *Template) error {
  tFromDb, err := r.readTemplateBase(id)
  if err != nil {
    return err
  }
  if tFromDb == nil {
    return fmt.Errorf("No template with id %v", id)
  }
  _, err = tx.Exec(`
    DELETE FROM template_parameter WHERE template_id = ?;
    DELETE FROM template_image WHERE template_id = ?;
    DELETE FROM template_text WHERE template_id = ?;
    UPDATE template SET name = ?, landscape = ? WHERE id = ?`,
    id, id, id,
    t.Name, t.Landscape, id)
  if err != nil {
    return fmt.Errorf("Couldn't update template data:\n%w", err)
  }
  t.Id = id
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
    INSERT INTO template_text(template_id, text, x, y, width, height)
    VALUES (?, ?, ?, ?, ?, ?)`)
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
    )
    if err != nil {
      return fmt.Errorf("Failed to insert parameter %v of text:\n%w", i, err)
    }
  }

  return nil
}
