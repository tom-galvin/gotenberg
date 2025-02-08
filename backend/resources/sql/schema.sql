CREATE TABLE IF NOT EXISTS template(
  id INTEGER PRIMARY KEY,
  uuid TEXT NOT NULL UNIQUE CHECK (LENGTH(uuid) = 36),
  name TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL,
  landscape INT NOT NULL, -- boolean
  min_size INT NOT NULL,
  max_size INT NOT NULL
);

CREATE TABLE IF NOT EXISTS template_parameter(
  id INTEGER PRIMARY KEY,
  template_id INT NOT NULL,
  name TEXT NOT NULL,
  max_length INT NOT NULL,
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
  height INT NOT NULL,
  font_id INT NOT NULL,
  font_size INT NOT NULL,
  FOREIGN KEY (template_id) REFERENCES template(id)
);

CREATE TABLE IF NOT EXISTS font(
  id INTEGER PRIMARY KEY,
  uuid TEXT NOT NULL UNIQUE CHECK (LENGTH(uuid) = 36),
  name TEXT NOT NULL,
  builtin_name TEXT,
  font_data BLOB
);

INSERT INTO font(id, uuid, name, builtin_name) VALUES
  (1, '4d98c9b6-8bb5-492d-9789-a2bb5ea8ab21', 'Go Regular', 'goregular'),
  (2, '703d9944-d746-431a-8df8-ecf61a1e5dad', 'Go Mono', 'gomono')
ON CONFLICT DO NOTHING;
