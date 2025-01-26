package template

import (
  "fmt"
  "errors"
	"database/sql"
)

type TemplateRepository struct {
  Db *sql.DB
}

func (r *TemplateRepository) readTemplateBase(id int) (*Template, error) {
  row := r.Db.QueryRow(`
    SELECT name, created_at, landscape
    FROM template
    WHERE id = ?`, id)

  t := Template{id:id}
  if err := row.Scan(&t.name, &t.createdAt, &t.landscape); err != nil {
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

  // select parameters
  paramRows, err := r.Db.Query(`
    SELECT id, name, max_length
    FROM template_parameter
    WHERE template_id = ?`, id)
  if err != nil {
    return nil, fmt.Errorf("Failed to query template parameters:\n%w", err)
  }
  defer paramRows.Close()

  for paramRows.Next() {
    var p Parameter
    if err := paramRows.Scan(&p.id, &p.name, &p.maxLength); err != nil {
      return nil, err
    }
    t.parameters = append(t.parameters, p)
  }

  if err = paramRows.Err(); err != nil {
    return nil, fmt.Errorf("Failed to read template parameters:\n%w", err)
  }

  // select images
  imageRows, err := r.Db.Query(`
    SELECT id, image, x, y, width, height
    FROM template_image
    WHERE template_id = ?`, id)
  if err != nil {
    return nil, fmt.Errorf("Failed to query template images:\n%w", err)
  }
  defer imageRows.Close()

  for imageRows.Next() {
    var i Image
    if err := imageRows.Scan(&i.id, &i.image, &i.x, &i.y, &i.width, &i.height); err != nil {
      return nil, err
    }
    t.images = append(t.images, i)
  }

  if err = imageRows.Err(); err != nil {
    return nil, fmt.Errorf("Failed to read template images:\n%w", err)
  }

  // select texts
  textRows, err := r.Db.Query(`
    SELECT id, text, x, y, width, height
    FROM template_text
    WHERE template_id = ?`, id)
  if err != nil {
    return nil, fmt.Errorf("Failed to query template texts:\n%w", err)
  }
  defer textRows.Close()

  for textRows.Next() {
    var tx Text
    if err := textRows.Scan(&tx.id, &tx.text, &tx.x, &tx.y, &tx.width, &tx.height); err != nil {
      return nil, err
    }
    t.texts = append(t.texts, tx)
  }

  if err = textRows.Err(); err != nil {
    return nil, fmt.Errorf("Failed to read template texts:\n%w", err)
  }

  return t, nil
}

func (r *TemplateRepository) Transact(f func (*sql.Tx) error) error {
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
    RETURNING id`, t.name, t.createdAt, t.landscape)
  if err := row.Scan(&t.id); err != nil {
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
    t.name, t.landscape, id)
  if err != nil {
    return fmt.Errorf("Couldn't update template data:\n%w", err)
  }
  t.id = id
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
  for i, p := range t.parameters {
    _, err := pStmt.Exec(t.id, p.name, sql.NullInt32{Int32:int32(p.maxLength),Valid:p.maxLength>0})
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
  for i, img := range t.images {
    _, err := iStmt.Exec(t.id,
      img.image,
      img.x, img.y,
      sql.NullInt32{Int32:int32(img.width),Valid:img.width>0},
      sql.NullInt32{Int32:int32(img.height),Valid:img.height>0},
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
  for i, txt := range t.texts {
    _, err := iStmt.Exec(t.id,
      txt.text,
      txt.x, txt.y,
      sql.NullInt32{Int32:int32(txt.width),Valid:txt.width>0},
      sql.NullInt32{Int32:int32(txt.height),Valid:txt.height>0},
    )
    if err != nil {
      return fmt.Errorf("Failed to insert parameter %v of text:\n%w", i, err)
    }
  }

  return nil
}
