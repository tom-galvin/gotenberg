CREATE TABLE IF NOT EXISTS print_queue(
  id INTEGER PRIMARY KEY,
  image_data BLOB NOT NULL
);

INSERT INTO print_queue(id, image_data) VALUES
  (1, 'hello!');
