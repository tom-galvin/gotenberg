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
		if err := paramRows.Scan(&p.Id, &p.Name, &p.MaxLength); err != nil {
			return nil, err
		}
		t.Parameters = append(t.Parameters, p)
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
		if err := imageRows.Scan(&i.Id, &i.Image, &i.X, &i.Y, &i.Width, &i.Height); err != nil {
			return nil, err
		}
		t.Images = append(t.Images, i)
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
		if err := textRows.Scan(&tx.Id, &tx.Text, &tx.X, &tx.Y, &tx.Width, &tx.Height); err != nil {
			return nil, err
		}
		t.Texts = append(t.Texts, tx)
	}

	if err = textRows.Err(); err != nil {
		return nil, fmt.Errorf("Failed to read template texts:\n%w", err)
	}

	return t, nil
}

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
		_, err := pStmt.Exec(t.Id, p.Name, sql.NullInt32{Int32: int32(p.MaxLength), Valid: p.MaxLength > 0})
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
			sql.NullInt32{Int32: int32(img.Width), Valid: img.Width > 0},
			sql.NullInt32{Int32: int32(img.Height), Valid: img.Height > 0},
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
		_, err := iStmt.Exec(t.Id,
			txt.Text,
			txt.X, txt.Y,
			sql.NullInt32{Int32: int32(txt.Width), Valid: txt.Width > 0},
			sql.NullInt32{Int32: int32(txt.Height), Valid: txt.Height > 0},
		)
		if err != nil {
			return fmt.Errorf("Failed to insert parameter %v of text:\n%w", i, err)
		}
	}

	return nil
}
