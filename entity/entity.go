package entity

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	// tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	git "github.com/libgit2/git2go/v34"
)

const (
	ProjectsPath = "refs/notes/ubik/projects"
	IssuesPath   = "refs/notes/ubik/issues"
	CommentsPath = "refs/notes/ubik/comments"
)

type Entity interface {
  GetRefPath() string
  GetId() string
  Marshal() ([]byte, error)
}

type Project struct {
	Id          string `json:"id"`
	Author      string `json:"author"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Closed      string `json:"closed"`
  RefPath     string `json:"refpath"`
	Progress    int    `json:"progress"`
  CreatedAt   time.Time `json:"created_at"`
  UpdatedAt   time.Time `json:"updated_at"`
}

func (p Project) GetRefPath() string {
  return ProjectsPath
}

func (p Project) GetId() string {
  return p.Id
}

func (p Project) Marshal() ([]byte, error) {
  return json.Marshal(p)
}

type Issue struct {
	Id          string `json:"id"`
	Author      string `json:"author"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Closed      string `json:"closed"`
	ParentType  string `json:"parent_type"`
	ParentId    string `json:"parent_id"`
  RefPath     string `json:"refpath"`
  CreatedAt   time.Time `json:"created_at"`
  UpdatedAt   time.Time `json:"updated_at"`
}

func (i Issue) GetRefPath() string {
  return IssuesPath
}

func (i Issue) GetId() string {
  return i.Id
}

func (i Issue) Marshal() ([]byte, error) {
  return json.Marshal(i)
}

type Comment struct {
	Id          string `json:"id"`
	Author      string `json:"author"`
	Description string `json:"content"`
	ParentType  string `json:"parent_type"`
	ParentId    string `json:"parent_id"`
  RefPath     string `json:"refpath"`
  CreatedAt   time.Time `json:"created_at"`
  UpdatedAt   time.Time `json:"updated_at"`
}

func (c Comment) GetRefPath() string {
  return CommentsPath
}

func (c Comment) GetId() string {
  return c.Id
}

func (c Comment) Marshal() ([]byte, error) {
  return json.Marshal(c)
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

func ListProjects() {
  refPath := ProjectsPath
  notes := GetNotes(refPath)
  uProjects := ProjectsFromGitNotes(notes)

  for _, uNotePtr := range uProjects {
    uNote := *uNotePtr

    issues := GetIssuesForProject(uNote.Id)

    fmt.Println("--------")
    fmt.Printf("Project %s\n", uNote.Id)
    fmt.Printf("Title: %s\n", uNote.Title)
    fmt.Printf("Description: %s\n", uNote.Description)
    fmt.Printf("Closed: %s\n", uNote.Closed)
    fmt.Println("Issues:")
    for _, issue := range issues {
      fmt.Printf("\t- %s (closed: %s)\n", issue.Title, issue.Closed)
      comments := GetCommentsForEntity(issue.Id)

      for _, comment := range comments {
        fmt.Printf("\t\t%s\n", comment.Description)
        fmt.Printf("\t\t- %s\n", comment.Author)
      }

    }
    fmt.Println("----------")
  }
}

func ListIssues() {
  refPath := IssuesPath
  notes := GetNotes(refPath)
  uNotes := IssuesFromGitNotes(notes)

  for _, uNotePtr := range uNotes {
    uNote := *uNotePtr
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

func ListComments() {
  refPath := CommentsPath
  notes := GetNotes(refPath)
  uNotes := CommentsFromGitNotes(notes)

  for _, uNotePtr := range uNotes {
    uNote := *uNotePtr
    fmt.Println("--------")
    fmt.Println(uNote.Description)
    fmt.Println(uNote.ParentId)
    fmt.Println()
  }
}

func GetCommentsForEntity(parentId string) []*Comment {
  refPath := CommentsPath
  notes := GetNotes(refPath)
  uNotes := CommentsFromGitNotes(notes)

  var filteredComments []*Comment

  for _, comment := range uNotes {
    if comment.ParentId == parentId {
      filteredComments = append(filteredComments, comment)
    }
  }

  return filteredComments
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

    newContent = string(newJSON)
  } else if err == nil {
    data := make(map[string]interface{})
    err := json.Unmarshal([]byte(note.Message()), &data)
    if err != nil {
      log.Fatalf("Failed to unmarshal data: %v\n", err)
    }
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
  return Add(entity)
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
		log.Fatalf("Failed to look up notes ref: %v", err)
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

func ProjectsFromGitNotes(gitNotes []*git.Note) []*Project {
	var uProjects []*Project
	for _, notePtr := range gitNotes {
		note := *notePtr
		author := *note.Author()
		lines := strings.Split(note.Message(), "\n")

		for _, line := range lines {
			if line != "" {
				var uProject Project
				err := json.Unmarshal([]byte(line), &uProject)
				if err != nil {
					log.Fatalf("Error unmarshaling JSON: %v", err)
				}

				uProject.Author = author.Email
				// fmt.Printf("%+v\n", uProject)
				uProjects = append(uProjects, &uProject)
			}
		}
	}

	return uProjects
}

func IssuesFromGitNotes(gitNotes []*git.Note) []*Issue {
	var uIssues []*Issue
	for _, notePtr := range gitNotes {
		note := *notePtr

    data := make(map[string]interface{})
    err := json.Unmarshal([]byte(note.Message()), &data)
    if err != nil {
      log.Fatalf("Failed to unmarshal data: %v\n", err)
    }

    for _, obj := range data {
      // TODO Need to implement entity.Unmarshal()
      obj := obj.(map[string]interface{})
      createdAt, _ := time.Parse(time.RFC3339, obj["created_at"].(string))
      updatedAt, _ := time.Parse(time.RFC3339, obj["updated_at"].(string))
      issue := Issue{
        Id: obj["id"].(string),
        Author: obj["author"].(string),
        Title: obj["title"].(string),
        Description: obj["description"].(string),
        ParentType: obj["parent_type"].(string),
        ParentId: obj["parent_id"].(string),
        RefPath: obj["refpath"].(string),
        CreatedAt: createdAt,
        UpdatedAt: updatedAt,
      }

      uIssues = append(uIssues, &issue)
    }
	}

	return uIssues
}

func CommentsFromGitNotes(gitNotes []*git.Note) []*Comment {
	var uComments []*Comment
	for _, notePtr := range gitNotes {
		note := *notePtr
		author := *note.Author()
		lines := strings.Split(note.Message(), "\n")

		for _, line := range lines {
			if line != "" {
				var uComment Comment
				err := json.Unmarshal([]byte(line), &uComment)
				if err != nil {
					log.Fatalf("Error unmarshaling JSON: %v", err)
				}

				uComment.Author = author.Email
				// fmt.Printf("%+v\n", uComment)
				uComments = append(uComments, &uComment)
			}
		}
	}

	return uComments
}
