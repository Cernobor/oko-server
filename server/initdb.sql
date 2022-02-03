DROP TABLE IF EXISTS users;
CREATE TABLE IF NOT EXISTS users (
  id integer PRIMARY KEY AUTOINCREMENT,
  name text NOT NULL UNIQUE
);
INSERT INTO users(id, name) VALUES(0, 'system');

DROP TABLE IF EXISTS features;
CREATE TABLE IF NOT EXISTS features (
  id integer PRIMARY KEY AUTOINCREMENT,
  owner_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name text NOT NULL,
  properties text NOT NULL,
  geom text NOT NULL
);

DROP TABLE IF EXISTS feature_photos;
CREATE TABLE IF NOT EXISTS feature_photos (
  id integer PRIMARY KEY AUTOINCREMENT,
  feature_id integer NOT NULL REFERENCES features(id) ON DELETE CASCADE,
  content_type text NOT NULL,
  file_contents blob NOT NULL
);