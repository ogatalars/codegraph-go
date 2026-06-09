package store

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// UpsertFile inserts or replaces a file record, returning its id.
func (s *Store) UpsertFile(path, pkg string, indexedAt int64) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO files(path, pkg, indexed_at) VALUES(?,?,?)
		 ON CONFLICT(path) DO UPDATE SET pkg=excluded.pkg, indexed_at=excluded.indexed_at`,
		path, pkg, indexedAt,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// InsertSymbol inserts a symbol, ignoring conflicts on fqn.
func (s *Store) InsertSymbol(sym InsertSymbolParams) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO symbols(file_id,name,fqn,kind,line,col,signature,docstring)
		 VALUES(?,?,?,?,?,?,?,?)`,
		sym.FileID, sym.Name, sym.FQN, sym.Kind, sym.Line, sym.Col, sym.Signature, sym.Docstring,
	)
	return err
}

// InsertEdge inserts an edge.
func (s *Store) InsertEdge(e InsertEdgeParams) error {
	_, err := s.db.Exec(
		`INSERT INTO edges(from_fqn,to_fqn,kind,file_id,line) VALUES(?,?,?,?,?)`,
		e.FromFQN, e.ToFQN, e.Kind, e.FileID, e.Line,
	)
	return err
}

// DeleteFileData removes all symbols and edges for a file (for re-index).
func (s *Store) DeleteFileData(fileID int64) error {
	if _, err := s.db.Exec(`DELETE FROM symbols WHERE file_id=?`, fileID); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM edges WHERE file_id=?`, fileID)
	return err
}

type InsertSymbolParams struct {
	FileID    int64
	Name      string
	FQN       string
	Kind      string
	Line      int
	Col       int
	Signature string
	Docstring string
}

type InsertEdgeParams struct {
	FromFQN string
	ToFQN   string
	Kind    string
	FileID  int64
	Line    int
}
