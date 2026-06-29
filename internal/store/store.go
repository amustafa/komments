package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type CursorPosition struct {
	Type string `json:"type"`
	Line int    `json:"line"`
	Col  int    `json:"col"`
}

type RangePosition struct {
	Type      string `json:"type"`
	StartLine int    `json:"start_line"`
	StartCol  int    `json:"start_col"`
	EndLine   int    `json:"end_line"`
	EndCol    int    `json:"end_col"`
}

type Comment struct {
	ID          int             `json:"id"`
	ProjectRoot string          `json:"project_root"`
	Timestamp   string          `json:"timestamp"`
	File        string          `json:"file"`
	Position    json.RawMessage `json:"position"`
	Text        string          `json:"text"`
	Archived    bool            `json:"archived"`
}

type Store struct {
	db          *sql.DB
	projectRoot string
}

func dbPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "komments", "comments.db")
}

func Open(projectRoot string) (*Store, error) {
	dbFile := dbPath()
	if err := os.MkdirAll(filepath.Dir(dbFile), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	for _, stmt := range []string{
		"PRAGMA journal_mode=WAL",
		`CREATE TABLE IF NOT EXISTS comments (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			project_root  TEXT    NOT NULL,
			timestamp     TEXT    NOT NULL,
			file          TEXT    NOT NULL,
			position      TEXT    NOT NULL,
			text          TEXT    NOT NULL,
			archived      INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_project
			ON comments(project_root, archived)`,
		`CREATE TABLE IF NOT EXISTS meta (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		"INSERT OR IGNORE INTO meta (key, value) VALUES ('schema_version', '2')",
	} {
		if _, err := db.Exec(stmt); err != nil {
			db.Close()
			return nil, fmt.Errorf("init db: %w", err)
		}
	}

	root := strings.TrimRight(projectRoot, "/")
	return &Store{db: db, projectRoot: root}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func scanComment(row interface{ Scan(...any) error }) (*Comment, error) {
	var c Comment
	var posJSON string
	var archived int
	err := row.Scan(&c.ID, &c.ProjectRoot, &c.Timestamp, &c.File, &posJSON, &c.Text, &archived)
	if err != nil {
		return nil, err
	}
	c.Position = json.RawMessage(posJSON)
	c.Archived = archived == 1
	return &c, nil
}

func (s *Store) scanRows(rows *sql.Rows) ([]*Comment, error) {
	var comments []*Comment
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (s *Store) ActiveComments() ([]*Comment, error) {
	rows, err := s.db.Query(
		"SELECT * FROM comments WHERE archived = 0 AND project_root = ? ORDER BY id",
		s.projectRoot,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanRows(rows)
}

func (s *Store) AllComments() ([]*Comment, error) {
	rows, err := s.db.Query(
		"SELECT * FROM comments WHERE project_root = ? ORDER BY id",
		s.projectRoot,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanRows(rows)
}

func (s *Store) GetComment(id int) (*Comment, error) {
	row := s.db.QueryRow("SELECT * FROM comments WHERE id = ?", id)
	return scanComment(row)
}

func (s *Store) AddComment(file string, position json.RawMessage, text string) (*Comment, error) {
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	result, err := s.db.Exec(
		"INSERT INTO comments (project_root, timestamp, file, position, text, archived) VALUES (?, ?, ?, ?, ?, 0)",
		s.projectRoot, ts, file, string(position), text,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &Comment{
		ID:          int(id),
		ProjectRoot: s.projectRoot,
		Timestamp:   ts,
		File:        file,
		Position:    position,
		Text:        text,
		Archived:    false,
	}, nil
}

func (s *Store) ArchiveComment(id int) (bool, error) {
	result, err := s.db.Exec("UPDATE comments SET archived = 1 WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func (s *Store) UnarchiveComment(id int) (bool, error) {
	result, err := s.db.Exec("UPDATE comments SET archived = 0 WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func (s *Store) UpdateComment(id int, newText string) (bool, error) {
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	result, err := s.db.Exec(
		"UPDATE comments SET text = ?, timestamp = ? WHERE id = ?",
		newText, ts, id,
	)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func (s *Store) DeleteComment(id int) (bool, error) {
	result, err := s.db.Exec("DELETE FROM comments WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}
