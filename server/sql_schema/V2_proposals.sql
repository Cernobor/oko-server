CREATE TABLE proposals (
  owner_id integer NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  description text NOT NULL,
  how text NOT NULL
);

PRAGMA user_version = 2;