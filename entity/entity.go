package entity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	// "github.com/blvrd/ubik/lens"
	"github.com/blvrd/ubik/shortcode"
	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

const (
	IssuesPath = "refs/notes/ubik/issues"
)

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

type Check struct {
	id        string
	Command   string
	Status    string
	CommitSHA string
}

type Project struct {
	id          string
	Author      string
	Title       string
	Description string
	Progress    int
	Issues      []*Issue
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   time.Time
}

type ProjectParams struct {
	Author      string
	Description string
}

type Memo struct {
	id        string
	Author    string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
	Comments  []Comment
}

type MemoParams struct {
	Author string
	Body   string
}

type Comment struct {
	id         string
	Author     string
	Body       string
	ParentType string
	ParentId   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  time.Time
}

type CommentParams struct {
	Author     string
	Body       string
	ParentType string
	ParentId   string
}

func NewComment(params CommentParams) Comment {
	return Comment{
		Author: params.Author,
		Body:   params.Body,
	}
}

type Issue struct {
	Id          string
	Author      string
	Title       string
	Description string
	ParentType  string
	ParentId    string
	RefPath     string
	shortcode   string
	ClosedAt    time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   time.Time
	Comments    []Comment
}

func NewIssue() Issue {
	author := GetAuthorEmail()
	return Issue{
		Author:      author,
		Title:       "",
		Description: "",
		ParentType:  "",
		ParentId:    "",
		RefPath:     IssuesPath,
	}
}

func (i Issue) IsPersisted() bool {
	return i.Id != ""
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
		ParentType  string    `json:"parent_type"`
		ParentId    string    `json:"parent_id"`
		RefPath     string    `json:"refpath"`
		Shortcode   string    `json:"shortcode"`
		ClosedAt    time.Time `json:"closed_at"`
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
		ClosedAt:    i.ClosedAt,
		ParentType:  i.ParentType,
		ParentId:    i.ParentId,
		RefPath:     i.RefPath,
		Shortcode:   i.shortcode,
		CreatedAt:   i.CreatedAt,
		UpdatedAt:   i.UpdatedAt,
		DeletedAt:   i.DeletedAt,
	}

	return json.Marshal(issueJSON)
}

func (i *Issue) UnmarshalJSON(data []byte) error {
	type IssueJSON struct {
		Id          string `json:"id"`
		Author      string `json:"author"`
		Title       string `json:"title"`
		Description string `json:"description"`
		ParentType  string `json:"parent_type"`
		ParentId    string `json:"parent_id"`
		RefPath     string `json:"refpath"`
		Shortcode   string `json:"shortcode"`
		ClosedAt    string `json:"closed_at"`
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

	i.Author = issueJSON.Author
	i.Title = issueJSON.Title
	i.Description = issueJSON.Description
	i.ParentType = issueJSON.ParentType
	i.ParentId = issueJSON.ParentId
	i.RefPath = issueJSON.RefPath
	i.shortcode = issueJSON.Shortcode

	closedAt, err := time.Parse(time.RFC3339, issueJSON.ClosedAt)
	if err != nil {
		closedAt = time.Time{}
	}
	createdAt, err := time.Parse(time.RFC3339, issueJSON.CreatedAt)
	if err != nil {
		createdAt = time.Time{}
	}
	updatedAt, err := time.Parse(time.RFC3339, issueJSON.UpdatedAt)
	if err != nil {
		updatedAt = time.Time{}
	}
	deletedAt, err := time.Parse(time.RFC3339, issueJSON.DeletedAt)
	if err != nil {
		deletedAt = time.Time{}
	}

	i.CreatedAt = createdAt
	i.UpdatedAt = updatedAt
	i.DeletedAt = deletedAt
	i.ClosedAt = closedAt

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
	i.ClosedAt = time.Time{}
	err := Update(i)
	if err != nil {
		return err
	}

	return nil
}

func (i *Issue) Close() error {
	i.ClosedAt = time.Now().UTC()
	err := Update(i)
	if err != nil {
		return err
	}

	return nil
}

func (i *Issue) CloseWithComment(message string) error {
	i.ClosedAt = time.Now().UTC()
	// TODO add a first class comment when closing an issue
	i.Description = fmt.Sprintf("%s\n\n%s", i.Description, message)
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
		"closed_at":   i.ClosedAt,
		"parent_type": i.ParentType,
		"parent_id":   i.ParentId,
		"refpath":     i.RefPath,
		"shortcode":   i.shortcode,
		"created_at":  i.CreatedAt,
		"updated_at":  i.UpdatedAt,
		"deleted_at":  i.DeletedAt,
		"comments":    i.Comments,
	}
}

func (i Issue) FilterValue() string {
	return i.Title
}

func GetWd() string {
	wd, err := os.Getwd()

	if err != nil {
		log.Fatalf("Failed to get current workding directory: %v", err)
	}

	return wd
}

func GetFirstCommit() string {
	cmd := exec.Command("git", "rev-list", "--max-parents=0", "HEAD")
	bytes, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	str := string(bytes)
	trimmed := strings.TrimSpace(str)
	return trimmed
}

func ListIssues() {
	refPath := IssuesPath
	notes, _ := GetNotes(refPath)
	uNotes := IssuesFromGitNotes(notes)

	for _, uNote := range uNotes {
		fmt.Println("--------")
		fmt.Println(uNote.Id)
		fmt.Println(uNote.Title)
		fmt.Println(uNote.Description)
		fmt.Println(uNote.ClosedAt)
		fmt.Println(uNote.ParentId)
		fmt.Println()
	}
}

func GetIssuesForProject(parentId string) []*Issue {
	refPath := IssuesPath
	notes, _ := GetNotes(refPath)
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

func Add(issue *Issue) error {
	if issue.IsPersisted() {
		log.Fatal("issue has already been persisted")
	}

	id := uuid.NewString()
	shortcodeCache := make(map[string]bool)
	shortcode := shortcode.GenerateShortcode(id, &shortcodeCache)
	issue.Id = id
	issue.shortcode = shortcode
	issue.Touch()

	jsonData, err := json.Marshal(issue)
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "hash-object", "--stdin", "-w")
	cmd.Stdin = bytes.NewReader(jsonData)

	b, err := cmd.Output()
	if err != nil {
		return err
	}

	hash := strings.TrimSpace(string(b))

	cmd = exec.Command("git", "update-ref", fmt.Sprintf("refs/ubik/issues/%s", issue.Id), hash)
	err = cmd.Run()

	if err != nil {
		log.Fatalf("%#v", err.Error())
		return err
	}

	return nil
}

func Update(issue *Issue) error {
	issue.Touch()
	jsonData, err := json.Marshal(issue)
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "hash-object", "--stdin", "-w")
	cmd.Stdin = bytes.NewReader(jsonData)

	b, err := cmd.Output()
	if err != nil {
		return err
	}

	hash := strings.TrimSpace(string(b))

	cmd = exec.Command("git", "update-ref", fmt.Sprintf("refs/ubik/issues/%s", issue.Id), hash)
	err = cmd.Run()

	if err != nil {
		log.Fatalf("%#v", err.Error())
		return err
	}

	return nil
}

func Remove(issue *Issue) error {
	err := issue.Delete()

	if err != nil {
		return err
	}

	return nil
}

func GetRefsByPath(refPath string) []string {
	cmd := exec.Command("git", "for-each-ref", "--format=%(objectname)", "refs/ubik")
	bytes, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	var noteIds []string
	str := string(bytes)
	for _, s := range strings.Split(str, "\n") {
		noteId := s
		if noteId != "" {
			noteIds = append(noteIds, noteId)
		}
	}

	return noteIds
}

func startsWithLinefeed(data []byte) bool {
	if len(data) > 0 && data[0] == '\n' {
		return true
	}
	return false
}

func GetNotes(refPath string) ([]Note, error) {
	refs := GetRefsByPath(refPath)

	var notes []Note
	for _, ref := range refs {
		cmd := exec.Command("git", "cat-file", "-p", ref)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			continue
		}

		note := Note{
			ObjectId: ref,
			Ref:      refPath,
			Message:  out.String(),
			Bytes:    out.Bytes(),
		}
		notes = append(notes, note)
	}

	return notes, nil
}

func IssuesFromGitNotes(gitNotes []Note) []*Issue {
	var issues []*Issue
	var closedIssues []*Issue
	for _, note := range gitNotes {
		var issue Issue
		err := json.Unmarshal(note.Bytes, &issue)
		if err != nil {
			log.Fatalf("Failed to unmarshal data: %#v\n", err)
		}

		if !issue.DeletedAt.IsZero() {
			continue
		}

		if !issue.ClosedAt.IsZero() {
			closedIssues = append(closedIssues, &issue)
			continue
		}
		var comments []Comment

		comment1 := NewComment(CommentParams{Author: "garrett@blvrd.co", Body: "This is a comment"})
		comment2 := NewComment(CommentParams{Author: "harsha@example.com", Body: "Another comment"})
		comment3 := NewComment(CommentParams{Author: "Code Bot", Body: "ATTENTION: I've detected a code smell"})
		comments = append(comments, comment1)
		comments = append(comments, comment2)
		comments = append(comments, comment3)
		issue.Comments = comments
		issues = append(issues, &issue)
	}

	sort.Sort(ByUpdatedAtDescending(issues))
	sort.Sort(ByUpdatedAtDescending(closedIssues))

	return append(issues, closedIssues...)
}
