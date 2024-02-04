package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	git "github.com/libgit2/git2go/v34"
)

type Memo struct {
	Id        string `json:"id"`
	Author    string `json:"author"`
	Content   string `json:"content"`
	Published string `json:"published"`
}

type Project struct {
	Id          string `json:"id"`
	Author      string `json:"author"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Complete    string `json:"complete"`
	Status      int    `json:"status"`
}

type Task struct {
	Id          string `json:"id"`
	Author      string `json:"author"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Complete    string `json:"complete"`
	ProjectId   string `json:"project_id"`
}

type Comment struct {
	Id         string `json:"id"`
	Author     string `json:"author"`
	Content    string `json:"content"`
	ParentType string `json:"parent_type"`
	ParentId   string `json:"parent_id"`
}

const (
	memosPath    = "refs/notes/ubik/memos"
	projectsPath = "refs/notes/ubik/projects"
	tasksPath    = "refs/notes/ubik/tasks"
	commentsPath = "refs/notes/ubik/comments"
)

func main() {
	refPath := GetRefPath()
	notes := GetNotes(refPath)
	uNotes := UbikNotesFromGitNotes(notes)

	for _, uNotePtr := range uNotes {
		uNote := *uNotePtr
		if uNote.Published == "true" {
			fmt.Println("--------")
			fmt.Println(uNote.Content)
			fmt.Println("\n")
		}
	}
}

func GetRefPath() string {
	configAuthor, err := exec.Command("git", "config", "user.email").Output()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	author := os.Getenv("GIT_AUTHOR_EMAIL")

	if author == "" {
		author = string(configAuthor)
	}

	refPath := fmt.Sprintf("refs/notes/ubik/memos/%s", author)
	sanitizedRefPath := strings.ReplaceAll(refPath, "\n", "")

	return sanitizedRefPath
}

func GetNotes(refPath string) []*git.Note {
	wd, _ := os.Getwd()

	repo, err := git.OpenRepository(wd)
	if err != nil {
		fmt.Printf("Failed to open repository: %v", err)
		os.Exit(1)
	}

	notesRefObj, err := repo.References.Lookup(refPath)
	if err != nil {
		fmt.Printf("Failed to look up notes ref: %v", err)
		os.Exit(1)
	}

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
			fmt.Printf("Error iterating notes: %v", err)
			os.Exit(1)
		}

		note, err := repo.Notes.Read(refPath, annotatedId)
		if err != nil {
			fmt.Printf("Error reading note: %v", err)
			os.Exit(1)
		}

		notes = append(notes, note)
	}

	return notes
}

func UbikNotesFromGitNotes(gitNotes []*git.Note) []*Memo {
	var uNotes []*Memo
	for _, notePtr := range gitNotes {
		note := *notePtr
		author := *note.Author()
		lines := strings.Split(note.Message(), "\n")

		for _, line := range lines {
			if line != "" {
				var uNote Memo
				err := json.Unmarshal([]byte(line), &uNote)
				if err != nil {
					fmt.Printf("Error unmarshaling JSON: %v", err)
					os.Exit(1)
				}

				uNote.Author = author.Email
				// fmt.Printf("%+v\n", uNote)
				uNotes = append(uNotes, &uNote)
			}
		}
	}

	return uNotes
}
