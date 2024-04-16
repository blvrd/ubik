package entity

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/blvrd/ubik/lens"
	"github.com/blvrd/ubik/shortcode"
	"github.com/charmbracelet/log"
	"github.com/google/uuid"
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
	IsPersisted() bool
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

func NewIssue() Issue {
	author := GetAuthorEmail()
	return Issue{
		Author:      author,
		Title:       "",
		Description: "",
		Closed:      "false",
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
		"shortcode":   i.shortcode,
		"created_at":  i.CreatedAt,
		"updated_at":  i.UpdatedAt,
		"deleted_at":  i.DeletedAt,
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
	log.Infof("str: %s", str)
	log.Infof("trimmed: %s", trimmed)
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
		fmt.Println(uNote.Closed)
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

	firstCommit := GetFirstCommit()

	var newContent string
	cmd := exec.Command("git", "notes", "--ref", IssuesPath, "show", firstCommit)
	note, err := cmd.CombinedOutput()
	log.Infof("output: %s", note)
	log.Infof("cmd: %s", cmd.String())
	log.Errorf("error: %v", err)
	id := uuid.NewString()
	shortcodeCache := make(map[string]bool)
	shortcode := shortcode.GenerateShortcode(id, &shortcodeCache)
	issue.Id = id
	issue.shortcode = shortcode

	if err != nil {
		data := make(map[string]interface{})

		data[id] = issue
		newJSON, err := json.Marshal(data)
		if err != nil {
			log.Fatalf("Failed to marshal entity: %v\n", err)
		}

		issue.Touch()
		newContent = string(newJSON)
	} else if err == nil {
		data := make(map[string]interface{})
		err := json.Unmarshal(note, &data)
		if err != nil {
			log.Fatalf("Failed to unmarshal data: %v\n", err)
		}
		issue.Touch()
		data[id] = issue

		newJSON, err := json.Marshal(data)

		if err != nil {
			log.Fatalf("Failed to marshal project: %v\n", err)
		}

		newContent = string(newJSON)
	}

	cmd = exec.Command("git", "notes", "--ref", IssuesPath, "add", "-m", newContent, "-f", firstCommit)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to add note to tree: %v", err)
	}

	return nil
}

func Update(entity Entity) error {
	firstCommit := GetFirstCommit()

	var newContent string
	cmd := exec.Command("git", "notes", "--ref", "refs/notes/ubik/issues", "show", firstCommit)
	note, err := cmd.Output()

	if err != nil {
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
		err := json.Unmarshal([]byte(note), &data)
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

	cmd = exec.Command("git", "notes", "--ref", "refs/notes/ubik/issues", "add", "-m", newContent, "-f", firstCommit)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to add note to tree: %v", err)
	}

	return nil
}

func Remove(entity Entity) error {
	firstCommit := GetFirstCommit()

	var newContent string
	cmd := exec.Command("git", "notes", "--ref", "refs/notes/ubik/issues", "show", firstCommit)
	note, err := cmd.Output()

	if err != nil {
		log.Fatalf("%v", err)
	} else if err == nil {
		data := make(map[string]interface{})
		err := json.Unmarshal([]byte(note), &data)
		if err != nil {
			log.Fatalf("Failed to unmarshal data: %v\n", err)
		}
		delete(data, entity.GetId())

		newJSON, err := json.Marshal(data)

		if err != nil {
			log.Fatalf("Failed to marshal project: %v\n", err)
		}

		newContent = string(newJSON)
	}

	cmd = exec.Command("git", "notes", "--ref", "refs/notes/ubik/issues", "add", "-m", newContent, "-f", firstCommit)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to add note to tree: %v", err)
	}

	return nil
}

func GetRefsByPath(refPath string) []string {
	// cmd := exec.Command("git", "for-each-ref", "--format='%(objectname)'", "refs/notes/ubik")
	cmd := exec.Command("git", "notes", "--ref", refPath, "list")
	bytes, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	var noteIds []string
	str := string(bytes)
	for _, s := range strings.Split(str, "\n") {
		noteId := strings.Split(s, " ")[0]
		if noteId != "" {
			noteIds = append(noteIds, noteId)
		}
	}

	log.Info(len(noteIds))
	return noteIds
}

func GetNotes(refPath string) ([]*Note, error) {
	// This function is probably where we want to implement lenses

	var notes []*Note
	refs := GetRefsByPath(refPath)

	cmd := exec.Command("git", "cat-file", "--batch")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	defer stdin.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	for _, object := range refs {
		fmt.Fprintln(stdin, object)
	}
	stdin.Close()

	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		objectInfo := strings.SplitN(scanner.Text(), " ", 3)
		objectHash := objectInfo[0]
		scanner.Scan()
		if scanner.Text() != "" {
			notes = append(notes, &Note{
				ObjectId: objectHash,
				Ref:      refPath,
				Message:  scanner.Text(),
        Bytes: scanner.Bytes(),
			})
		}
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return notes, nil
}

func IssuesFromGitNotes(gitNotes []*Note) []*Issue {
	var issues []*Issue
	var closedIssues []*Issue
	for _, notePtr := range gitNotes {
		note := *notePtr

    b, err := os.ReadFile("entity/issue_lenses.json")
    if err != nil {
      panic(err)
    }
    lensSource := lens.NewLensSource(b)
    newBytes := lens.ApplyLensToDoc(lensSource, note.Bytes)
    log.Debugf("new doc: %#v", string(newBytes))

		data := make(map[string]Issue)
		err = json.Unmarshal(note.Bytes, &data)
		if err != nil {
			log.Info("note message: %s", note.Message)
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

	sort.Sort(ByUpdatedAtDescending(issues))
	sort.Sort(ByUpdatedAtDescending(closedIssues))

	return append(issues, closedIssues...)
}
