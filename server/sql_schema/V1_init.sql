CREATE TABLE users (
  id integer PRIMARY KEY AUTOINCREMENT,
  name text NOT NULL UNIQUE
);
INSERT INTO users(id, name) VALUES(0, 'system');

CREATE TABLE features (
  id integer PRIMARY KEY AUTOINCREMENT,
  owner_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name text NOT NULL,
  deadline text,
  properties text NOT NULL,
  geom text NOT NULL
);

CREATE TABLE feature_photos (
  id integer PRIMARY KEY AUTOINCREMENT,
  feature_id integer NOT NULL REFERENCES features(id) ON DELETE CASCADE,
  thumbnail_content_type text NOT NULL,
  content_type text NOT NULL,
  thumbnail_contents blob NOT NULL,
  contents blob NOT NULL
);

PRAGMA user_version = 1;