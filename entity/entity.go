package entity

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	git "github.com/libgit2/git2go/v34"
)

const (
	IssuesPath = "refs/notes/ubik/issues"
)

type Entity interface {
	GetRefPath() string
	GetId() string
	ToMap() map[string]interface{}
	Touch()
	Delete() error
	Restore() error
	json.Marshaler
	json.Unmarshaler
}

type Listable interface {
	FilterValue() string
}

type ByUpdatedAtDescending []*Issue

func (n ByUpdatedAtDescending) Len() int           { return len(n) }
func (n ByUpdatedAtDescending) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n ByUpdatedAtDescending) Less(i, j int) bool { return n[i].UpdatedAt.After(n[j].UpdatedAt) }

type ByUpdatedAtAscending []*Issue

func (n ByUpdatedAtAscending) Len() int           { return len(n) }
func (n ByUpdatedAtAscending) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n ByUpdatedAtAscending) Less(i, j int) bool { return n[i].UpdatedAt.Before(n[j].UpdatedAt) }

type Issue struct {
	Id          string
	Author      string
	Title       string
	Description string
	Closed      string
	ParentType  string
	ParentId    string
	RefPath     string
	shortcode   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   time.Time
}

func NewIssue(
	author string,
	title string,
	description string,
	parentType string,
	parentId string,
) Issue {
	id := uuid.NewString()
	shortcode := GenerateShortcode(id)
	closed := "false"

	return Issue{
		Id:          id,
		Author:      author,
		Title:       title,
		Description: description,
		Closed:      closed,
		shortcode:   shortcode,
		ParentType:  parentType,
		ParentId:    parentId,
		RefPath:     IssuesPath,
	}
}

func (i Issue) Shortcode() string {
	return i.shortcode
}

func (i Issue) GetRefPath() string {
	return IssuesPath
}

func (i Issue) GetId() string {
	return i.Id
}

func (i Issue) MarshalJSON() ([]byte, error) {
	type IssueJSON struct {
		Id          string    `json:"id"`
		Author      string    `json:"author"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Closed      string    `json:"closed"`
		ParentType  string    `json:"parent_type"`
		ParentId    string    `json:"parent_id"`
		RefPath     string    `json:"refpath"`
		Shortcode   string    `json:"shortcode"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		DeletedAt   time.Time `json:"deleted_at"`
	}

	// Convert the original struct to the custom struct
	issueJSON := IssueJSON{
		Id:          i.Id,
		Author:      i.Author,
		Title:       i.Title,
		Description: i.Description,
		Closed:      i.Closed,
		ParentType:  i.ParentType,
		ParentId:    i.ParentId,
		RefPath:     i.RefPath,
		Shortcode:   i.shortcode,
		CreatedAt:   i.CreatedAt,
		UpdatedAt:   i.UpdatedAt,
		DeletedAt:   i.DeletedAt,
	}

	log.Info("marshaling issue!")
	return json.Marshal(issueJSON)
}

func (i *Issue) UnmarshalJSON(data []byte) error {
	type IssueJSON struct {
		Id          string `json:"id"`
		Author      string `json:"author"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Closed      string `json:"closed"`
		ParentType  string `json:"parent_type"`
		ParentId    string `json:"parent_id"`
		RefPath     string `json:"refpath"`
		Shortcode   string `json:"shortcode"`
		CreatedAt   string `json:"created_at"`
		UpdatedAt   string `json:"updated_at"`
		DeletedAt   string `json:"deleted_at"`
	}

	var issueJSON IssueJSON
	err := json.Unmarshal(data, &issueJSON)
	if err != nil {
		return err
	}

	i.Id = issueJSON.Id
  log.Infof("unmarshaling issue: %s", i.Id)

	i.Author = issueJSON.Author
	i.Title = issueJSON.Title
	i.Description = issueJSON.Description
	i.Closed = issueJSON.Closed
	i.ParentType = issueJSON.ParentType
	i.ParentId = issueJSON.ParentId
	i.RefPath = issueJSON.RefPath
	i.shortcode = issueJSON.Shortcode

	createdAt, err := time.Parse(time.RFC3339, issueJSON.CreatedAt)
	if err != nil {
    i.CreatedAt = time.Time{}
    return nil
	}
	updatedAt, err := time.Parse(time.RFC3339, issueJSON.UpdatedAt)
	if err != nil {
    i.UpdatedAt = time.Time{}
    return nil
	}
	deletedAt, err := time.Parse(time.RFC3339, issueJSON.DeletedAt)
	if err != nil {
    i.DeletedAt = time.Time{}
    return nil
	}

	i.CreatedAt = createdAt
	i.UpdatedAt = updatedAt
	i.DeletedAt = deletedAt

	return nil
}

func (i *Issue) Touch() {
	timestamp := time.Now().UTC()
	if time.Time.IsZero(i.CreatedAt) {
		i.CreatedAt = timestamp
	}
	i.UpdatedAt = timestamp
}

func (i *Issue) Delete() error {
	i.DeletedAt = time.Now().UTC()
	err := Update(i)
	if err != nil {
		return err
	}

	return nil
}

func (i *Issue) Restore() error {
	i.DeletedAt = time.Time{}
	err := Update(i)
	if err != nil {
		return err
	}

	return nil
}

func (i *Issue) Open() error {
	i.Closed = "false"
	err := Update(i)
	if err != nil {
		return err
	}

	return nil
}

func (i *Issue) Close() error {
	i.Closed = "true"
	err := Update(i)
	if err != nil {
		return err
	}

	return nil
}

func (i Issue) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":          i.Id,
		"author":      i.Author,
		"title":       i.Title,
		"description": i.Description,
		"closed":      i.Closed,
		"parent_type": i.ParentType,
		"parent_id":   i.ParentId,
		"refpath":     i.RefPath,
		"created_at":  i.CreatedAt,
		"updated_at":  i.UpdatedAt,
		"deleted_at":  i.DeletedAt,
	}
}

func (i Issue) FilterValue() string {
	return i.Title
}

const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
const shortcodeLength = 6

// Placeholder for your actual storage check
func isUnique(shortcode string) bool {
	// Implement logic to check if shortcode exists in your storage
	return true // Assume it's unique for this example
}

// Placeholder for your actual storage operation
func storeShortcode(shortcode, uuid string) {
	// Implement logic to store the shortcode-UUID mapping
}

func GenerateShortcode(uuid string) string {
	hash := sha256.Sum256([]byte(uuid))
	hashInt := new(big.Int).SetBytes(hash[:])

	base := big.NewInt(int64(len(charset)))
	var shortcode string

	for {
		shortcode = ""
		tempHashInt := new(big.Int).Set(hashInt)
		for i := 0; i < shortcodeLength; i++ {
			mod := new(big.Int)
			tempHashInt.DivMod(tempHashInt, base, mod)
			shortcode = string(charset[mod.Int64()]) + shortcode
		}

		if isUnique(shortcode) {
			storeShortcode(shortcode, uuid) // Save the unique shortcode
			break
		} else {
			// Modify the hashInt (e.g., by adding 1) to try a different shortcode
			hashInt = hashInt.Add(hashInt, big.NewInt(1))
		}
	}

	return shortcode
}

func GetWd() string {
	wd, err := os.Getwd()

	if err != nil {
		log.Fatalf("Failed to get current workding directory: %v", err)
	}

	return wd
}

func GetFirstCommit(repo *git.Repository) *git.Commit {
	revWalk, err := repo.Walk()
	if err != nil {
		log.Fatalf("Failed to create revision walker: %v\n", err)
	}
	defer revWalk.Free()

	// Start from the HEAD
	err = revWalk.PushHead()
	if err != nil {
		log.Fatalf("Failed to start rev walk at HEAD: %v\n", err)
	}

	revWalk.Sorting(git.SortTime)

	// Iterating to find the first commit
	var firstCommit *git.Commit
	oid := new(git.Oid)
	for revWalk.Next(oid) == nil {
		commit, err := repo.LookupCommit(oid)
		if err != nil {
			log.Fatalf("Failed to lookup commit: %v\n", err)
		}
		// Assuming the first commit we can reach is the oldest/root
		firstCommit = commit
	}

	if firstCommit == nil {
		log.Fatalf("No commits found in repository.")
	}

	return firstCommit
}

func ListIssues() {
	refPath := IssuesPath
	notes := GetNotes(refPath)
	uNotes := IssuesFromGitNotes(notes)

	for _, uNote := range uNotes {
		fmt.Println("--------")
		fmt.Println(uNote.Id)
		fmt.Println(uNote.Title)
		fmt.Println(uNote.Description)
		fmt.Println(uNote.Closed)
		fmt.Println(uNote.ParentId)
		fmt.Println()
	}
}

func GetIssuesForProject(parentId string) []*Issue {
	refPath := IssuesPath
	notes := GetNotes(refPath)
	uNotes := IssuesFromGitNotes(notes)

	var filteredIssues []*Issue

	for _, issue := range uNotes {
		if issue.ParentId == parentId {
			filteredIssues = append(filteredIssues, issue)
		}
	}

	return filteredIssues
}

func GetAuthorEmail() string {
	configAuthor, err := exec.Command("git", "config", "user.email").Output()
	if err != nil {
		log.Fatal(err)
	}

	author := os.Getenv("GIT_AUTHOR_EMAIL")

	if author == "" {
		author = string(configAuthor)
	}

	return strings.TrimSpace(author)
}

func Add(entity Entity) error {
	wd := GetWd()
	repo, err := git.OpenRepository(wd)
	if err != nil {
		return fmt.Errorf("Failed to open repository: %v", err)
	}

	firstCommit := GetFirstCommit(repo)

	var newContent string
	note, err := repo.Notes.Read(entity.GetRefPath(), firstCommit.Id())
	if err != nil && git.IsErrorCode(err, git.ErrNotFound) {
		data := make(map[string]interface{})
		data[entity.GetId()] = entity
		newJSON, err := json.Marshal(data)
		if err != nil {
			log.Fatalf("Failed to marshal entity: %v\n", err)
		}

		entity.Touch()
		newContent = string(newJSON)
	} else if err == nil {
		data := make(map[string]interface{})
		err := json.Unmarshal([]byte(note.Message()), &data)
		if err != nil {
			log.Fatalf("Failed to unmarshal data: %v\n", err)
		}
		entity.Touch()
		data[entity.GetId()] = entity

		newJSON, err := json.Marshal(data)

		if err != nil {
			log.Fatalf("Failed to marshal project: %v\n", err)
		}

		newContent = string(newJSON)
	} else {
		return err
	}

	sig, err := repo.DefaultSignature()
	if err != nil {
		return fmt.Errorf("Couldn't find default signature: %v", err)
	}

	_, err = repo.Notes.Create(
		entity.GetRefPath(),
		sig,
		sig,
		firstCommit.Id(),
		newContent,
		true,
	)
	if err != nil {
		return fmt.Errorf("Failed to add note to tree: %v", err)
	}

	return nil
}

func Update(entity Entity) error {
	wd := GetWd()
	repo, err := git.OpenRepository(wd)
	if err != nil {
		return fmt.Errorf("Failed to open repository: %v", err)
	}

	firstCommit := GetFirstCommit(repo)

	var newContent string
	note, err := repo.Notes.Read(entity.GetRefPath(), firstCommit.Id())
	if err != nil && git.IsErrorCode(err, git.ErrNotFound) {
		data := make(map[string]interface{})
		entity.Touch()
		data[entity.GetId()] = entity
		newJSON, err := json.Marshal(data)
		if err != nil {
			log.Fatalf("Failed to marshal entity: %v\n", err)
		}

		newContent = string(newJSON)
	} else if err == nil {
		data := make(map[string]interface{})
		err := json.Unmarshal([]byte(note.Message()), &data)
		if err != nil {
			log.Fatalf("Failed to unmarshal data: %v\n", err)
		}
		entity.Touch()
		data[entity.GetId()] = entity

		newJSON, err := json.Marshal(data)

		if err != nil {
			log.Fatalf("Failed to marshal project: %v\n", err)
		}

		newContent = string(newJSON)
	} else {
		return err
	}

	sig, err := repo.DefaultSignature()
	if err != nil {
		return fmt.Errorf("Couldn't find default signature: %v", err)
	}

	_, err = repo.Notes.Create(
		entity.GetRefPath(),
		sig,
		sig,
		firstCommit.Id(),
		newContent,
		true,
	)
	if err != nil {
		return fmt.Errorf("Failed to add note to tree: %v", err)
	}

	return nil
}

func Remove(entity Entity) error {
	wd := GetWd()
	repo, err := git.OpenRepository(wd)
	if err != nil {
		return fmt.Errorf("Failed to open repository: %v", err)
	}

	firstCommit := GetFirstCommit(repo)

	var newContent string
	note, err := repo.Notes.Read(entity.GetRefPath(), firstCommit.Id())
	if err != nil && git.IsErrorCode(err, git.ErrNotFound) {
		log.Fatalf("%v", err)
	} else if err == nil {
		data := make(map[string]interface{})
		err := json.Unmarshal([]byte(note.Message()), &data)
		if err != nil {
			log.Fatalf("Failed to unmarshal data: %v\n", err)
		}
		delete(data, entity.GetId())

		newJSON, err := json.Marshal(data)

		if err != nil {
			log.Fatalf("Failed to marshal project: %v\n", err)
		}

		newContent = string(newJSON)
	} else {
		return err
	}

	sig, err := repo.DefaultSignature()
	if err != nil {
		return fmt.Errorf("Couldn't find default signature: %v", err)
	}

	_, err = repo.Notes.Create(
		entity.GetRefPath(),
		sig,
		sig,
		firstCommit.Id(),
		newContent,
		true,
	)
	if err != nil {
		return fmt.Errorf("Failed to add note to tree: %v", err)
	}

	return nil
}

func OpenRepo(wd string) *git.Repository {
	repo, err := git.OpenRepository(wd)
	if err != nil {
		log.Fatalf("Failed to open repository: %v", err)
	}

	return repo
}

func GetRefsByPath(repo *git.Repository, refPath string) *git.Reference {
	notesRefObj, err := repo.References.Lookup(refPath)
	if err != nil {
		log.Errorf("Failed to look up notes ref: %v", err)
		log.Infof("Creating ref: %v", err)

		firstCommit := GetFirstCommit(repo)
		sig, err := repo.DefaultSignature()
		if err != nil {
			log.Errorf("Couldn't find default signature: %v", err)
		}

		_, err = repo.Notes.Create(
			IssuesPath,
			sig,
			sig,
			firstCommit.Id(),
			"{}",
			true,
		)
		notesRefObj, err := repo.References.Lookup(refPath)
		return notesRefObj
	}

	return notesRefObj
}

func GetNotes(refPath string) []*git.Note {
	wd := GetWd()
	repo := OpenRepo(wd)
	notesRefObj := GetRefsByPath(repo, refPath)

	iter, err := repo.NewNoteIterator(notesRefObj.Name())
	if err != nil {
		fmt.Printf("Failed to get notes iterator: %v", err)
	}
	defer iter.Free()

	var notes []*git.Note

	var annotatedId *git.Oid
	for {
		_, annotatedId, err = iter.Next()
		if err != nil {
			if git.IsErrorCode(err, git.ErrIterOver) {
				break // End of the iterator
			}
			log.Fatalf("Error iterating notes: %v", err)
		}

		note, err := repo.Notes.Read(refPath, annotatedId)
		if err != nil {
			log.Fatalf("Error reading note: %v", err)
		}

		notes = append(notes, note)
	}

	return notes
}

func IssuesFromGitNotes(gitNotes []*git.Note) []*Issue {
	var issues []*Issue
	var closedIssues []*Issue
	for _, notePtr := range gitNotes {
		note := *notePtr

		data := make(map[string]Issue)
		err := json.Unmarshal([]byte(note.Message()), &data)
		if err != nil {
			log.Fatalf("Failed to unmarshal data: %v\n", err)
		}

		for _, issue := range data {
      issue := issue
			if !issue.DeletedAt.IsZero() {
				continue
			}

			if issue.Closed == "true" {
				closedIssues = append(closedIssues, &issue)
				continue
			}

			issues = append(issues, &issue)
		}
	}

  log.Infof("%+v", issues)


	sort.Sort(ByUpdatedAtDescending(issues))
	sort.Sort(ByUpdatedAtDescending(closedIssues))

	return append(issues, closedIssues...)
}
