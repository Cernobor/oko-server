DROP TABLE IF EXISTS users;
CREATE TABLE IF NOT EXISTS users (
  id integer PRIMARY KEY AUTOINCREMENT,
  name text NOT NULL UNIQUE
);
INSERT INTO users(id, name) VALUES(0, 'system');

DROP TABLE IF EXISTS features;
CREATE TABLE IF NOT EXISTS features (
  id integer PRIMARY KEY AUTOINCREMENT,
  owner_id integer,
  name text NOT NULL,
  description text,
  category text,
  geom text NOT NULL,
  FOREIGN KEY(owner_id) REFERENCES users(id) ON DELETE CASCADE
);
