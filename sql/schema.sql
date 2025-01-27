CREATE TABLE IF NOT EXISTS template(
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL,
  landscape INT NOT NULL -- boolean
);

CREATE TABLE IF NOT EXISTS template_parameter(
  id INTEGER PRIMARY KEY,
  template_id INT NOT NULL,
  name TEXT NOT NULL,
  max_length INT,
  FOREIGN KEY (template_id) REFERENCES template(id)
);

CREATE TABLE IF NOT EXISTS template_image(
  id INTEGER PRIMARY KEY,
  template_id INT NOT NULL,
  image BLOB NOT NULL,
  x INT NOT NULL,
  y INT NOT NULL,
  width INT NOT NULL,
  height INT NOT NULL,
  FOREIGN KEY (template_id) REFERENCES template(id)
);

CREATE TABLE IF NOT EXISTS template_text(
  id INTEGER PRIMARY KEY,
  template_id INT NOT NULL,
  text TEXT NOT NULL,
  x INT NOT NULL,
  y INT NOT NULL,
  width INT NOT NULL,
  height INT,
  FOREIGN KEY (template_id) REFERENCES template(id)
);
