package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mattn/go-sqlite3"
)

func registerSQLiteExtensions() {
	sql.Register("sqlite3_with_extensions", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			err := conn.CreateModule("refs", &referencesModule{})
			if err != nil {
				return err
			}

			conn.CreateModule("commits", &commitsModule{})
			if err != nil {
				return err
			}

			return nil
		},
	})
}

func queryForCommits() []Commit {
	var commits []Commit
	db, err := sql.Open("sqlite3_with_extensions", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("create virtual table commits using commits(hash, message, author_name, author_email, timestamp)")
	if err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query("select hash, message, author_email, timestamp from commits")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var hash, message, authorEmail, timestampStr string
		rows.Scan(&hash, &message, &authorEmail, &timestampStr)
		timestamp, err := time.Parse(object.DateFormat, timestampStr)
		if err != nil {
			panic(err)
		}
		commits = append(commits, Commit{
			Hash:        hash,
			Message:     strings.TrimSuffix(message, "\n"),
			AuthorEmail: authorEmail,
			Timestamp:   timestamp,
		})
	}

	return commits
}

type commitsModule struct {
}

func (m *commitsModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %s (
			hash TEXT,
      message TEXT,
      author_name TEXT,
      author_email TEXT,
      timestamp TEXT
		)`, args[0]))
	if err != nil {
		return nil, err
	}
	return &commitsTable{}, nil
}

func (m *commitsModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *commitsModule) DestroyModule() {}

type commitsTable struct {
	commits []Commit
}

func (v *commitsTable) Open() (sqlite3.VTabCursor, error) {
	var commits []*object.Commit

	repo, err := git.PlainOpen(".")

	if err != nil {
		panic(err)
	}

	logOptions := git.LogOptions{
		Order: git.LogOrderCommitterTime,
	}

	gitCommits, err := repo.Log(&logOptions)

	if err != nil {
		panic(err)
	}

	err = gitCommits.ForEach(func(c *object.Commit) error {
		commits = append(commits, c)
		return nil
	})

	if err != nil {
		panic(err)
	}

	return &commitCursor{0, commits}, nil
}

func (v *commitsTable) BestIndex(csts []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	used := make([]bool, len(csts))
	return &sqlite3.IndexResult{
		IdxNum: 0,
		IdxStr: "default",
		Used:   used,
	}, nil
}

func (v *commitsTable) Disconnect() error { return nil }
func (v *commitsTable) Destroy() error    { return nil }

type commitCursor struct {
	index   int
	commits []*object.Commit
}

func (vc *commitCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	switch col {
	case 0:
		c.ResultText(vc.commits[vc.index].Hash.String())
	case 1:
		c.ResultText(vc.commits[vc.index].Message)
	case 2:
		c.ResultText(vc.commits[vc.index].Author.Name)
	case 3:
		c.ResultText(vc.commits[vc.index].Author.Email)
	case 4:
		timestamp := vc.commits[vc.index].Author.When
		c.ResultText(timestamp.Format(object.DateFormat))
	}
	return nil
}

func (vc *commitCursor) Filter(idxNum int, idxStr string, vals []any) error {
	vc.index = 0
	return nil
}

func (vc *commitCursor) Next() error {
	vc.index++
	return nil
}

func (vc *commitCursor) EOF() bool {
	return vc.index >= len(vc.commits)
}

func (vc *commitCursor) Rowid() (int64, error) {
	return int64(vc.index), nil
}

func (vc *commitCursor) Close() error {
	return nil
}

func queryForReferences(query string) []Ref {
	var references []Ref
	db, err := sql.Open("sqlite3_with_extensions", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("create virtual table refs using refs(hash, name)")
	if err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var hash, name string
		rows.Scan(&hash, &name)
		if err != nil {
			panic(err)
		}

		references = append(references, Ref{
			Hash: hash,
			Name: name,
		})
	}

	return references
}

type referencesModule struct {
}

func (m *referencesModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %s (
      hash TEXT,
      name TEXT
		)`, args[0]))
	if err != nil {
		return nil, err
	}
	return &referencesTable{}, nil
}

func (m *referencesModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	return m.Create(c, args)
}

func (m *referencesModule) DestroyModule() {}

type referencesTable struct {
	references []*plumbing.Reference
}

func (v *referencesTable) Open() (sqlite3.VTabCursor, error) {
	var references []*plumbing.Reference

	repo, err := git.PlainOpen(".")

	if err != nil {
		panic(err)
	}

	refs, err := repo.References()

	if err != nil {
		panic(err)
	}

	err = refs.ForEach(func(r *plumbing.Reference) error {
		references = append(references, r)
		return nil
	})

	if err != nil {
		panic(err)
	}

	return &referenceCursor{0, references}, nil
}

func (v *referencesTable) BestIndex(csts []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	used := make([]bool, len(csts))
	return &sqlite3.IndexResult{
		IdxNum: 0,
		IdxStr: "default",
		Used:   used,
	}, nil
}

func (v *referencesTable) Disconnect() error { return nil }
func (v *referencesTable) Destroy() error    { return nil }

type referenceCursor struct {
	index      int
	references []*plumbing.Reference
}

func (vc *referenceCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	switch col {
	case 0:
		c.ResultText(vc.references[vc.index].Hash().String())
	case 1:
		c.ResultText(vc.references[vc.index].Name().String())
	}
	return nil
}

func (vc *referenceCursor) Filter(idxNum int, idxStr string, vals []any) error {
	vc.index = 0
	return nil
}

func (vc *referenceCursor) Next() error {
	vc.index++
	return nil
}

func (vc *referenceCursor) EOF() bool {
	return vc.index >= len(vc.references)
}

func (vc *referenceCursor) Rowid() (int64, error) {
	return int64(vc.index), nil
}

func (vc *referenceCursor) Close() error {
	return nil
}
