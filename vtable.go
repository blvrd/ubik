package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
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

			err = conn.CreateModule("commits", &commitsModule{})
			if err != nil {
				return err
			}

			err = conn.CreateModule("blobs", &blobsModule{})
			if err != nil {
				return err
			}

			return nil
		},
	})
}

func queryForCommits(query string, db *sql.DB) []Commit {
	var commits []Commit

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var hash, message, authorEmail, timestampStr string
		err = rows.Scan(&hash, &message, &authorEmail, &timestampStr)
		if err != nil {
			panic(err)
		}
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

func queryForReferences(query string, db *sql.DB) []Ref {
	var references []Ref

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var hash, name string
		err = rows.Scan(&hash, &name)
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

func queryForBlobs(query string, db *sql.DB) []Blob {
	var blobs []Blob

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var hash string
		var size int64
		var content []byte
		err = rows.Scan(&hash, &size, &content)
		if err != nil {
			panic(err)
		}

		blobs = append(blobs, Blob{
			Hash:    hash,
			Size:    size,
			Content: content,
		})
	}

	return blobs
}

type blobsModule struct {
}

func (m *blobsModule) Create(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	err := c.DeclareVTab(fmt.Sprintf(`
		CREATE TABLE %s (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      hash TEXT,
      size INTEGER,
      content BLOB
		)`, args[0]))
	if err != nil {
		return nil, err
	}
	return &blobsTable{}, nil
}

func (m *blobsModule) Connect(c *sqlite3.SQLiteConn, args []string) (sqlite3.VTab, error) {
	log.Debugf("🪚 args: %#v", args)
	return m.Create(c, args)
}

func (m *blobsModule) DestroyModule() {}

type blobsTable struct {
	blobs []*object.Blob
}

func (v *blobsTable) Open() (sqlite3.VTabCursor, error) {
	var blobs []*object.Blob

	repo, err := git.PlainOpen(".")

	if err != nil {
		panic(err)
	}

	objs, err := repo.BlobObjects()

	if err != nil {
		panic(err)
	}

	err = objs.ForEach(func(b *object.Blob) error {
		blobs = append(blobs, b)
		return nil
	})

	if err != nil {
		panic(err)
	}

	return &blobCursor{0, blobs}, nil
}

func (v *blobsTable) BestIndex(csts []sqlite3.InfoConstraint, ob []sqlite3.InfoOrderBy) (*sqlite3.IndexResult, error) {
	used := make([]bool, len(csts))
	return &sqlite3.IndexResult{
		IdxNum: 0,
		IdxStr: "default",
		Used:   used,
	}, nil
}

func (v *blobsTable) Disconnect() error  { return nil }
func (v *blobsTable) Destroy() error     { return nil }
func (v *blobsTable) Delete(x any) error { panic("trying to delete") }
func (v *blobsTable) Insert(id any, vals []any) (int64, error) {
  str, ok := vals[3].(string)
  if !ok {
    return 0, errors.New("value is not a string")
  }
  jsonData := []byte(str)
	repo, _ := git.PlainOpen(".")
  obj := repo.Storer.NewEncodedObject()
  obj.SetType(plumbing.BlobObject)
  obj.SetSize(int64(len(jsonData)))
  writer, err := obj.Writer()
  if err != nil {
    debug("%#v", err.Error())
    return 0, err
  }
  _, err = writer.Write(jsonData)
  if err != nil {
    debug("%#v", err.Error())
    return 0, err
  }
  err = writer.Close()
  if err != nil {
    debug("%#v", err.Error())
    return 0, err
  }
  hash, err := repo.Storer.SetEncodedObject(obj)
  if err != nil {
    debug("%#v", err.Error())
    return 0, err
  }

  blob, err := object.GetBlob(repo.Storer, hash)
  if err != nil {
    debug("%#v", err.Error())
    return 0, err
  }

  v.blobs = append(v.blobs, blob)

	return int64(len(v.blobs)), nil
}

func (v *blobsTable) Update(any, []any) error { panic("trying to update") }

type blobCursor struct {
	index int
	blobs []*object.Blob
}

func (vc *blobCursor) Column(c *sqlite3.SQLiteContext, col int) error {
	switch col {
	case 0:
    rowid, _ := vc.Rowid()
    c.ResultInt64(rowid)
	case 1:
    c.ResultText(vc.blobs[vc.index].Hash.String())
	case 2:
    c.ResultInt64(vc.blobs[vc.index].Size)
  case 3:
    reader, err := vc.blobs[vc.index].Reader()
    if err != nil {
      panic(err)
    }

    b, err := io.ReadAll(reader)

    if err != nil {
      panic(err)
    }

    c.ResultBlob(b)
	}
	return nil
}

func (vc *blobCursor) Filter(idxNum int, idxStr string, vals []any) error {
	vc.index = 0
	return nil
}

func (vc *blobCursor) Next() error {
	vc.index++
	return nil
}

func (vc *blobCursor) EOF() bool {
	return vc.index >= len(vc.blobs)
}

func (vc *blobCursor) Rowid() (int64, error) {
	return int64(vc.index), nil
}

func (vc *blobCursor) Close() error {
	return nil
}
