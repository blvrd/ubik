package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	git "github.com/libgit2/git2go/v34"
	"github.com/spf13/cobra"
  "github.com/google/uuid"

)

type Memo struct {
	Id          string `json:"id"`
	Author      string `json:"author"`
	Title       string `json:"title"`
	Description string `json:"content"`
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
	// ========================
	// CLI Commands
	// ========================

	// rootCmd represents the base command when called without any subcommands
	var rootCmd = &cobra.Command{
		Use:   "ubik",
		Short: "A brief description of your application",
		// Run: func(cmd *cobra.Command, args []string) { },
	}

  var termUiCmd = &cobra.Command{
    Use: "termui",
    Short: "Use Ubik from the handy Terminal UI",
    Run: func(cmd *cobra.Command, args []string) { fmt.Println("heyyyy from the TUI") },
  }

	var memosCmd = &cobra.Command{
		Use:   "memos",
		Short: "Memos are notes to yourself or other contributors.",
	}

	var memosListCmd = &cobra.Command{
		Use:   "list",
		Short: "List memos you've written",
		Run: func(cmd *cobra.Command, args []string) { ListMemos() },
	}

	var memosAddCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a new thing",
		Run: func(cmd *cobra.Command, args []string) {
      titleFlag, _ := cmd.Flags().GetString("title")
			descriptionFlag, _ := cmd.Flags().GetString("description")

      AddMemo(titleFlag, descriptionFlag)
		},
	}

	memosAddCmd.Flags().StringP(
		"title",
    "t",
		"",
		"Title for the memo",
	)

	memosAddCmd.Flags().StringP(
		"description",
    "d",
		"",
		"description for the memo",
	)

  memosAddCmd.MarkFlagRequired("title")
  memosAddCmd.MarkFlagRequired("description")

	var projectsCmd = &cobra.Command{
		Use:   "projects",
		Short: "projects",
	}

	var projectsListCmd = &cobra.Command{
		Use:   "list",
		Short: "List projects you've created",
		Run: func(cmd *cobra.Command, args []string) { ListProjects() },
	}

	var projectsAddCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a new thing",
		Run: func(cmd *cobra.Command, args []string) {
      titleFlag, _ := cmd.Flags().GetString("title")
			descriptionFlag, _ := cmd.Flags().GetString("description")

      AddProject(titleFlag, descriptionFlag)
		},
	}

	projectsAddCmd.Flags().StringP(
		"title",
    "t",
		"",
		"Title for the project",
	)

	projectsAddCmd.Flags().StringP(
		"description",
    "d",
		"",
		"Description for the project",
	)

  projectsAddCmd.MarkFlagRequired("title")
  projectsAddCmd.MarkFlagRequired("description")

	var tasksCmd = &cobra.Command{
		Use:   "tasks",
		Short: "tasks",
	}

	var tasksListCmd = &cobra.Command{
		Use:   "list",
		Short: "List tasks you've created",
		Run: func(cmd *cobra.Command, args []string) { ListTasks() },
	}

	var tasksAddCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a new thing",
		Run: func(cmd *cobra.Command, args []string) {
      titleFlag, _ := cmd.Flags().GetString("title")
			descriptionFlag, _ := cmd.Flags().GetString("description")
      projectIdFlag, _ := cmd.Flags().GetString("project_id")

      AddTask(titleFlag, descriptionFlag, projectIdFlag)
		},
	}

	tasksAddCmd.Flags().String(
		"title",
		"",
		"Title for the task",
	)

	tasksAddCmd.Flags().String(
		"description",
		"",
		"Description for the task",
	)

	tasksAddCmd.Flags().String(
		"project_id",
		"",
		"Project ID for the task",
	)

  tasksAddCmd.MarkFlagRequired("title")
  tasksAddCmd.MarkFlagRequired("description")

  rootCmd.AddCommand(memosCmd, projectsCmd, tasksCmd, termUiCmd)
	memosCmd.AddCommand(memosAddCmd, memosListCmd)
  projectsCmd.AddCommand(projectsAddCmd, projectsListCmd)
  tasksCmd.AddCommand(tasksAddCmd, tasksListCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func ListMemos() {
  refPath := memosPath
  notes := GetNotes(refPath)
  uNotes := MemosFromGitNotes(notes)

  for _, uNotePtr := range uNotes {
    uNote := *uNotePtr
    fmt.Println("--------")
    fmt.Println(uNote.Title)
    fmt.Println(uNote.Description)
    fmt.Println()
  }
}

func AddMemo(title, description string) {
  wd := GetWd()

  repo, err := git.OpenRepository(wd)
  if err != nil {
    fmt.Printf("Failed to open repository: %v\n", err)
    os.Exit(1)
  }

  revWalk, err := repo.Walk()
  if err != nil {
    fmt.Printf("Failed to create revision walker: %v\n", err)
    os.Exit(1)
  }
  defer revWalk.Free()

  // Start from the HEAD
  err = revWalk.PushHead()
  if err != nil {
    fmt.Printf("Failed to start rev walk at HEAD: %v\n", err)
    os.Exit(1)
  }

  revWalk.Sorting(git.SortTime)

  // Iterating to find the first commit
  var firstCommit *git.Commit
  oid := new(git.Oid)
  for revWalk.Next(oid) == nil {
    commit, err := repo.LookupCommit(oid)
    if err != nil {
      fmt.Printf("Failed to lookup commit: %v\n", err)
      os.Exit(1)
    }
    // Assuming the first commit we can reach is the oldest/root
    firstCommit = commit
  }

  if firstCommit == nil {
    fmt.Println("No commits found in repository.")
    os.Exit(1)
  }

  // Getting the root tree of the first commit
  rootTree, err := firstCommit.Tree()
  if err != nil {
    fmt.Printf("Failed to get root tree: %v\n", err)
    os.Exit(1)
  }

  // Constructing the Memo struct
  memo := Memo{
    Id:          uuid.New().String(),
    Author:      GetAuthorEmail(),
    Title:       title,
    Description: description,
  }

  memoBytes, err := json.Marshal(memo)
  if err != nil {
    fmt.Printf("Failed to marshal memo: %v\n", err)
    os.Exit(1)
  }

  var newContent string

  note, err := repo.Notes.Read(memosPath, rootTree.Id())
  if err != nil && !git.IsErrorCode(err, git.ErrNotFound) {
    newContent = string(memoBytes)
  } else if err == nil {
    newContent = note.Message() + "\n" + string(memoBytes)
  }

  sig, err := repo.DefaultSignature()
  if err != nil {
    fmt.Printf("Couldn't find default signature: %v\n", err)
    os.Exit(1)
  }

  // Explicitly create a note attached to the tree. Note that
  // this usage is unconventional and might not be supported by Git interfaces.
  _, err = repo.Notes.Create(
    memosPath,
    sig,
    sig,
    rootTree.Id(),
    newContent,
    true,
  )
  if err != nil {
    fmt.Printf("Failed to add note to tree: %v\n", err)
    os.Exit(1)
  }

  fmt.Println("Memo added successfully to the root tree of the first commit.")
}

func ListProjects() {
  refPath := projectsPath
  notes := GetNotes(refPath)
  uProjects := ProjectsFromGitNotes(notes)

  for _, uNotePtr := range uProjects {
    uNote := *uNotePtr

    tasks := GetTasksForProject(uNote.Id)

    fmt.Println("--------")
    fmt.Printf("Project %s\n", uNote.Id)
    fmt.Printf("Title: %s\n", uNote.Title)
    fmt.Printf("Description: %s\n", uNote.Description)
    fmt.Printf("Complete: %s\n", uNote.Complete)
    fmt.Println("Tasks:")
    for _, task := range tasks {
      fmt.Printf("\t- %s (complete: %s)\n", task.Title, task.Complete)
    }
    fmt.Println("----------")
  }
}

func AddProject(title, description string) {
  wd := GetWd()

  repo, err := git.OpenRepository(wd)
  if err != nil {
    fmt.Printf("Failed to open repository: %v\n", err)
    os.Exit(1)
  }

  revWalk, err := repo.Walk()
  if err != nil {
    fmt.Printf("Failed to create revision walker: %v\n", err)
    os.Exit(1)
  }
  defer revWalk.Free()

  // Start from the HEAD
  err = revWalk.PushHead()
  if err != nil {
    fmt.Printf("Failed to start rev walk at HEAD: %v\n", err)
    os.Exit(1)
  }

  revWalk.Sorting(git.SortTime)

  // Iterating to find the first commit
  var firstCommit *git.Commit
  oid := new(git.Oid)
  for revWalk.Next(oid) == nil {
    commit, err := repo.LookupCommit(oid)
    if err != nil {
      fmt.Printf("Failed to lookup commit: %v\n", err)
      os.Exit(1)
    }
    // Assuming the first commit we can reach is the oldest/root
    firstCommit = commit
  }

  if firstCommit == nil {
    fmt.Println("No commits found in repository.")
    os.Exit(1)
  }

  // Getting the root tree of the first commit
  rootTree, err := firstCommit.Tree()
  if err != nil {
    fmt.Printf("Failed to get root tree: %v\n", err)
    os.Exit(1)
  }

  // Constructing the Memo struct
  memo := Project{
    Id:          uuid.New().String(),
    Author:      GetAuthorEmail(), // Make sure you define this
    Title:       title,
    Description: description,
    Complete:    "false",
  }

  memoBytes, err := json.Marshal(memo)
  if err != nil {
    fmt.Printf("Failed to marshal memo: %v\n", err)
    os.Exit(1)
  }

  var newContent string

  note, err := repo.Notes.Read(projectsPath, rootTree.Id())
  if err != nil && !git.IsErrorCode(err, git.ErrNotFound) {
    newContent = string(memoBytes)
  } else if err == nil {
    newContent = note.Message() + "\n" + string(memoBytes)
  }

  sig, err := repo.DefaultSignature()
  if err != nil {
    fmt.Printf("Couldn't find default signature: %v\n", err)
    os.Exit(1)
  }

  // Explicitly create a note attached to the tree. Note that
  // this usage is unconventional and might not be supported by Git interfaces.
  _, err = repo.Notes.Create(
    projectsPath,
    sig,
    sig,
    rootTree.Id(),
    newContent,
    true,
  )
  if err != nil {
    fmt.Printf("Failed to add note to tree: %v\n", err)
    os.Exit(1)
  }

  fmt.Println("Project added successfully to the root tree of the first commit.")
}

func ListTasks() {
  refPath := tasksPath
  notes := GetNotes(refPath)
  uNotes := TasksFromGitNotes(notes)

  for _, uNotePtr := range uNotes {
    uNote := *uNotePtr
    fmt.Println("--------")
    fmt.Println(uNote.Title)
    fmt.Println(uNote.Description)
    fmt.Println(uNote.Complete)
    fmt.Println(uNote.ProjectId)
    fmt.Println()
  }
}

func GetTasksForProject(projectId string) []*Task {
  refPath := tasksPath
  notes := GetNotes(refPath)
  uNotes := TasksFromGitNotes(notes)

  var filteredTasks []*Task

  for _, task := range uNotes {
    if task.ProjectId == projectId {
      filteredTasks = append(filteredTasks, task)
    }
  }

  return filteredTasks
}

func AddTask(title, description, projectId string) {
  wd := GetWd()

  repo, err := git.OpenRepository(wd)
  if err != nil {
    fmt.Printf("Failed to open repository: %v\n", err)
    os.Exit(1)
  }

  revWalk, err := repo.Walk()
  if err != nil {
    fmt.Printf("Failed to create revision walker: %v\n", err)
    os.Exit(1)
  }
  defer revWalk.Free()

  // Start from the HEAD
  err = revWalk.PushHead()
  if err != nil {
    fmt.Printf("Failed to start rev walk at HEAD: %v\n", err)
    os.Exit(1)
  }

  revWalk.Sorting(git.SortTime)

  // Iterating to find the first commit
  var firstCommit *git.Commit
  oid := new(git.Oid)
  for revWalk.Next(oid) == nil {
    commit, err := repo.LookupCommit(oid)
    if err != nil {
      fmt.Printf("Failed to lookup commit: %v\n", err)
      os.Exit(1)
    }
    // Assuming the first commit we can reach is the oldest/root
    firstCommit = commit
  }

  if firstCommit == nil {
    fmt.Println("No commits found in repository.")
    os.Exit(1)
  }

  // Getting the root tree of the first commit
  rootTree, err := firstCommit.Tree()
  if err != nil {
    fmt.Printf("Failed to get root tree: %v\n", err)
    os.Exit(1)
  }

  // Constructing the task struct
  task := Task{
    Id:          uuid.New().String(),
    Author:      GetAuthorEmail(), // Make sure you define this
    Title:       title,
    Description: description,
    Complete:    "false",
    ProjectId:   projectId,
  }

  taskBytes, err := json.Marshal(task)
  if err != nil {
    fmt.Printf("Failed to marshal task: %v\n", err)
    os.Exit(1)
  }

  var newContent string

  note, err := repo.Notes.Read(tasksPath, rootTree.Id())
  if err != nil && !git.IsErrorCode(err, git.ErrNotFound) {
    newContent = string(taskBytes)
  } else if err == nil {
    newContent = note.Message() + "\n" + string(taskBytes)
  }

  sig, err := repo.DefaultSignature()
  if err != nil {
    fmt.Printf("Couldn't find default signature: %v\n", err)
    os.Exit(1)
  }

  // Explicitly create a note attached to the tree. Note that
  // this usage is unconventional and might not be supported by Git interfaces.
  _, err = repo.Notes.Create(
    tasksPath,
    sig,
    sig,
    rootTree.Id(),
    newContent,
    true,
  )
  if err != nil {
    fmt.Printf("Failed to add note to tree: %v\n", err)
    os.Exit(1)
  }

  fmt.Println("Task added successfully to the root tree of the first commit.")
}

func GetAuthorEmail() string {
	configAuthor, err := exec.Command("git", "config", "user.email").Output()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	author := os.Getenv("GIT_AUTHOR_EMAIL")

	if author == "" {
		author = string(configAuthor)
	}

  return author
}

func GetWd() string {
  wd, err := os.Getwd()

  if err != nil {
		fmt.Printf("Failed to get current workding directory: %v", err)
		os.Exit(1)
  }

  return wd
}

func OpenRepo(wd string) *git.Repository {
	repo, err := git.OpenRepository(wd)
	if err != nil {
		fmt.Printf("Failed to open repository: %v", err)
		os.Exit(1)
	}

  return repo
}

func GetRefsByPath(repo *git.Repository, refPath string) *git.Reference {
	notesRefObj, err := repo.References.Lookup(refPath)
	if err != nil {
		fmt.Printf("Failed to look up notes ref: %v", err)
		os.Exit(1)
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

func MemosFromGitNotes(gitNotes []*git.Note) []*Memo {
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
					fmt.Printf("Error unmarshaling JSON: %v", err)
					os.Exit(1)
				}

				uProject.Author = author.Email
				// fmt.Printf("%+v\n", uProject)
				uProjects = append(uProjects, &uProject)
			}
		}
	}

	return uProjects
}

func TasksFromGitNotes(gitNotes []*git.Note) []*Task {
	var uTasks []*Task
	for _, notePtr := range gitNotes {
		note := *notePtr
		author := *note.Author()
		lines := strings.Split(note.Message(), "\n")

		for _, line := range lines {
			if line != "" {
				var uTask Task
				err := json.Unmarshal([]byte(line), &uTask)
				if err != nil {
					fmt.Printf("Error unmarshaling JSON: %v", err)
					os.Exit(1)
				}

				uTask.Author = author.Email
				// fmt.Printf("%+v\n", uTask)
				uTasks = append(uTasks, &uTask)
			}
		}
	}

	return uTasks
}
