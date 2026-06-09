package indexer

type Symbol struct {
	FileID    int64
	Name      string
	FQN       string
	Kind      string // func | method | type | var | const | interface | struct
	Line      int
	Col       int
	Signature string
	Docstring string
}

type Edge struct {
	FromFQN string
	ToFQN   string
	Kind    string // call | implements | embeds | references
	FileID  int64
	Line    int
}
