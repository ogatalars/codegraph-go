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

// --- write methods ---

func (s *Store) UpsertFile(path, pkg string, indexedAt int64) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO files(path, pkg, indexed_at) VALUES(?,?,?)
		 ON CONFLICT(path) DO UPDATE SET pkg=excluded.pkg, indexed_at=excluded.indexed_at
		 RETURNING id`,
		path, pkg, indexedAt,
	)
	if err != nil {
		// fallback: query after upsert
		var id int64
		if e2 := s.db.QueryRow(`SELECT id FROM files WHERE path=?`, path).Scan(&id); e2 != nil {
			return 0, err
		}
		return id, nil
	}
	return res.LastInsertId()
}

func (s *Store) InsertSymbol(sym InsertSymbolParams) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO symbols(file_id,name,fqn,kind,line,col,signature,docstring)
		 VALUES(?,?,?,?,?,?,?,?)`,
		sym.FileID, sym.Name, sym.FQN, sym.Kind, sym.Line, sym.Col, sym.Signature, sym.Docstring,
	)
	return err
}

func (s *Store) InsertEdge(e InsertEdgeParams) error {
	_, err := s.db.Exec(
		`INSERT INTO edges(from_fqn,to_fqn,kind,file_id,line) VALUES(?,?,?,?,?)`,
		e.FromFQN, e.ToFQN, e.Kind, e.FileID, e.Line,
	)
	return err
}

func (s *Store) DeleteFileData(fileID int64) error {
	if _, err := s.db.Exec(`DELETE FROM symbols WHERE file_id=?`, fileID); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM edges WHERE file_id=?`, fileID)
	return err
}

// --- read methods ---

func (s *Store) SearchSymbols(pattern, kind string, limit int) ([]SymbolRow, error) {
	if limit <= 0 {
		limit = 20
	}
	q := `SELECT s.id, s.file_id, s.name, s.fqn, s.kind, s.line, s.col,
	             COALESCE(s.signature,''), COALESCE(s.docstring,''), f.path
	      FROM symbols s JOIN files f ON f.id = s.file_id
	      WHERE s.name LIKE ?`
	args := []any{"%" + pattern + "%"}
	if kind != "" {
		q += " AND s.kind = ?"
		args = append(args, kind)
	}
	q += " LIMIT ?"
	args = append(args, limit)
	return s.scanSymbols(q, args...)
}

func (s *Store) GetSymbol(fqn string) (*SymbolRow, error) {
	q := `SELECT s.id, s.file_id, s.name, s.fqn, s.kind, s.line, s.col,
	             COALESCE(s.signature,''), COALESCE(s.docstring,''), f.path
	      FROM symbols s JOIN files f ON f.id = s.file_id
	      WHERE s.fqn = ?`
	rows, err := s.scanSymbols(q, fqn)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

func (s *Store) GetCallers(fqn string) ([]EdgeRow, error) {
	return s.scanEdges(`SELECT id,from_fqn,to_fqn,kind,file_id,line FROM edges WHERE to_fqn=?`, fqn)
}

func (s *Store) GetCallees(fqn string) ([]EdgeRow, error) {
	return s.scanEdges(`SELECT id,from_fqn,to_fqn,kind,file_id,line FROM edges WHERE from_fqn=?`, fqn)
}

func (s *Store) GetFiles(prefix string) ([]FileRow, error) {
	rows, err := s.db.Query(`SELECT id,path,pkg,indexed_at FROM files WHERE path LIKE ? ORDER BY path`, prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []FileRow
	for rows.Next() {
		var r FileRow
		if err := rows.Scan(&r.ID, &r.Path, &r.Pkg, &r.IndexedAt); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (s *Store) GetStatus() (files, symbols, edges int, err error) {
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&files)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM symbols`).Scan(&symbols)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM edges`).Scan(&edges)
	return
}

// --- helpers ---

func (s *Store) scanSymbols(q string, args ...any) ([]SymbolRow, error) {
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []SymbolRow
	for rows.Next() {
		var r SymbolRow
		if err := rows.Scan(&r.ID, &r.FileID, &r.Name, &r.FQN, &r.Kind, &r.Line, &r.Col, &r.Signature, &r.Docstring, &r.FilePath); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (s *Store) scanEdges(q string, args ...any) ([]EdgeRow, error) {
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []EdgeRow
	for rows.Next() {
		var r EdgeRow
		if err := rows.Scan(&r.ID, &r.FromFQN, &r.ToFQN, &r.Kind, &r.FileID, &r.Line); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// --- param / result types ---

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

type SymbolRow struct {
	ID        int64
	FileID    int64
	Name      string
	FQN       string
	Kind      string
	Line      int
	Col       int
	Signature string
	Docstring string
	FilePath  string
}

type EdgeRow struct {
	ID      int64
	FromFQN string
	ToFQN   string
	Kind    string
	FileID  int64
	Line    int
}

type FileRow struct {
	ID        int64
	Path      string
	Pkg       string
	IndexedAt int64
}
