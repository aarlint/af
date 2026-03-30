package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS presidents (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  term INTEGER NOT NULL,
  start_date TEXT NOT NULL,
  end_date TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS actions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  president TEXT NOT NULL REFERENCES presidents(id),
  eo TEXT,
  title TEXT NOT NULL,
  date TEXT NOT NULL,
  category TEXT NOT NULL,
  url TEXT
);

CREATE TABLE IF NOT EXISTS impacts (
  action_id INTEGER NOT NULL REFERENCES actions(id),
  country TEXT NOT NULL,
  score INTEGER NOT NULL CHECK(score BETWEEN -2 AND 2),
  reason TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (action_id, country)
);

CREATE INDEX IF NOT EXISTS idx_actions_president ON actions(president);
CREATE INDEX IF NOT EXISTS idx_actions_category ON actions(category);
CREATE INDEX IF NOT EXISTS idx_actions_date ON actions(date);
CREATE INDEX IF NOT EXISTS idx_impacts_country ON impacts(country);
`

var defaultPresidents = []struct {
	ID, Name, Start, End string
	Term                 int
}{
	{"obama1", "Barack Obama", "2009-01-20", "2013-01-20", 1},
	{"obama2", "Barack Obama", "2013-01-20", "2017-01-20", 2},
	{"trump1", "Donald Trump", "2017-01-20", "2021-01-20", 1},
	{"biden", "Joe Biden", "2021-01-20", "2025-01-20", 1},
	{"trump2", "Donald Trump", "2025-01-20", "2029-01-20", 2},
}

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Performance pragmas
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-8000",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("pragma %s: %w", pragma, err)
		}
	}

	return db, nil
}

func Init(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	// Seed presidents
	stmt, err := db.Prepare("INSERT OR IGNORE INTO presidents (id, name, term, start_date, end_date) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("prepare presidents: %w", err)
	}
	defer stmt.Close()

	for _, p := range defaultPresidents {
		if _, err := stmt.Exec(p.ID, p.Name, p.Term, p.Start, p.End); err != nil {
			return fmt.Errorf("insert president %s: %w", p.ID, err)
		}
	}

	return nil
}
