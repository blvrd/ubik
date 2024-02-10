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

type Project struct {
	Id          string `json:"id"`
	Author      string `json:"author"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Closed      string `json:"closed"`
	Progress    int    `json:"progress"`
}

type Issue struct {
	Id          string `json:"id"`
	Author      string `json:"author"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Closed    string `json:"closed"`
	ParentType  string `json:"parent_type"`
	ParentId    string `json:"parent_id"`
}

type Comment struct {
	Id          string `json:"id"`
	Author      string `json:"author"`
	Description string `json:"content"`
	ParentType  string `json:"parent_type"`
	ParentId    string `json:"parent_id"`
}

const (
	projectsPath = "refs/notes/ubik/projects"
	issuesPath   = "refs/notes/ubik/issues"
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

  var pushCmd = &cobra.Command{
    Use: "push",
    Short: "push",
    Run: func(cmd *cobra.Command, args []string) { fmt.Println("pushing ubik refs to remote:") },
  }

  var pullCmd = &cobra.Command{
    Use: "pull",
    Short: "pull",
    Run: func(cmd *cobra.Command, args []string) { fmt.Println("pulling ubik refs from remote:") },
  }

	var projectsCmd = &cobra.Command{
		Use:   "projects",
		Short: "projects",
	}

	var projectsListCmd = &cobra.Command{
		Use:   "list",
		Short: "List projects you've created",
		Run: func(cmd *cobra.Command, args []string) { ListProjects() },
	}

  var nukeCmd = &cobra.Command{
		Use:   "nuke",
		Short: "Nuke data - use for debugging purposes",
		Run: func(cmd *cobra.Command, args []string) {
      Nuke()
    },
  }

	var projectsAddCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a new thing",
    PreRunE: func(cmd *cobra.Command, args[]string) error {
      titleFlag, _ := cmd.Flags().GetString("title")
			descriptionFlag, _ := cmd.Flags().GetString("description")
      termUiFlag, _ := cmd.Flags().GetBool("termui")

      if !termUiFlag {
        if titleFlag == "" || descriptionFlag == "" {
          return fmt.Errorf("if --termui is not set, then --title and --description must be set.")
        }
      }

      return nil
    },
		Run: func(cmd *cobra.Command, args []string) {
      titleFlag, _ := cmd.Flags().GetString("title")
			descriptionFlag, _ := cmd.Flags().GetString("description")
      termUiFlag, _ := cmd.Flags().GetBool("termui")

      if termUiFlag {
        os.Exit(0)
      } else {
        AddProject(titleFlag, descriptionFlag)
      }
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

	projectsAddCmd.Flags().Bool(
    "termui",
    false,
    "Open the terminal UI",
  )

  projectsAddCmd.MarkFlagsRequiredTogether("title", "description")

	var issuesCmd = &cobra.Command{
		Use:   "issues",
		Short: "issues",
	}

	var issuesListCmd = &cobra.Command{
		Use:   "list",
		Short: "List issues you've created",
		Run: func(cmd *cobra.Command, args []string) { ListIssues() },
	}

	var issuesAddCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a new thing",
		Run: func(cmd *cobra.Command, args []string) {
      titleFlag, _ := cmd.Flags().GetString("title")
			descriptionFlag, _ := cmd.Flags().GetString("description")
      parentIdFlag, _ := cmd.Flags().GetString("parent_id")
      parentTypeFlag, _ := cmd.Flags().GetString("parentType")

      AddIssue(titleFlag, descriptionFlag, parentIdFlag, parentTypeFlag)
		},
	}

	issuesAddCmd.Flags().String(
		"title",
		"",
		"Title for the issue",
	)

	issuesAddCmd.Flags().String(
		"description",
		"",
		"Description for the issue",
	)

	issuesAddCmd.Flags().String(
		"project_id",
		"",
		"Project ID for the issue",
	)

  issuesAddCmd.MarkFlagRequired("title")
  issuesAddCmd.MarkFlagRequired("description")

  rootCmd.AddCommand(projectsCmd, issuesCmd, termUiCmd, pushCmd, pullCmd, nukeCmd)
  projectsCmd.AddCommand(projectsAddCmd, projectsListCmd)
  issuesCmd.AddCommand(issuesAddCmd, issuesListCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func Nuke() {
  exec.Command("./ubik_clear_all").Run()
}

func GetFirstCommit(repo *git.Repository) *git.Commit {
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

  return firstCommit
}

func GetTree(commit *git.Commit) *git.Tree {
  // Getting the root tree of the first commit
  tree, err := commit.Tree()
  if err != nil {
    fmt.Printf("Failed to get root tree: %v\n", err)
    os.Exit(1)
  }

  return tree
}

func ListProjects() {
  refPath := projectsPath
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

  // Constructing the project struct
  project := Project{
    Id:          uuid.New().String(),
    Author:      GetAuthorEmail(), // Make sure you define this
    Title:       title,
    Description: description,
    Closed:    "false",
  }

  projectBytes, err := json.Marshal(project)
  if err != nil {
    fmt.Printf("Failed to marshal project: %v\n", err)
    os.Exit(1)
  }

  var newContent string

  note, err := repo.Notes.Read(projectsPath, rootTree.Id())
  if err != nil && !git.IsErrorCode(err, git.ErrNotFound) {
    newContent = string(projectBytes)
  } else if err == nil {
    newContent = note.Message() + "\n" + string(projectBytes)
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

func ListIssues() {
  refPath := issuesPath
  notes := GetNotes(refPath)
  uNotes := IssuesFromGitNotes(notes)

  for _, uNotePtr := range uNotes {
    uNote := *uNotePtr
    fmt.Println("--------")
    fmt.Println(uNote.Title)
    fmt.Println(uNote.Description)
    fmt.Println(uNote.Closed)
    fmt.Println(uNote.ParentId)
    fmt.Println()
  }
}

func GetIssuesForProject(parentId string) []*Issue {
  refPath := issuesPath
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

func AddIssue(title, description, parentId, parentType string) {
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
  fmt.Printf("%+v", rootTree)
  if err != nil {
    fmt.Printf("Failed to get root tree: %v\n", err)
    os.Exit(1)
  }

  // Constructing the issue struct
  issue := Issue{
    Id:          uuid.New().String(),
    Author:      GetAuthorEmail(), // Make sure you define this
    Title:       title,
    Description: description,
    Closed:      "false",
    ParentId:    parentId,
  }

  issueBytes, err := json.Marshal(issue)
  if err != nil {
    fmt.Printf("Failed to marshal issue: %v\n", err)
    os.Exit(1)
  }

  var newContent string

  note, err := repo.Notes.Read(issuesPath, rootTree.Id())
  if err != nil && git.IsErrorCode(err, git.ErrNotFound) {
    newContent = string(issueBytes)
  } else if err == nil {
    newContent = note.Message() + "\n" + string(issueBytes)
  } else {
    fmt.Printf("%v\n", err)
  }

  sig, err := repo.DefaultSignature()
  if err != nil {
    fmt.Printf("Couldn't find default signature: %v\n", err)
    os.Exit(1)
  }

  fmt.Printf("%s\n", newContent)
  // Explicitly create a note attached to the tree. Note that
  // this usage is unconventional and might not be supported by Git interfaces.
  _, err = repo.Notes.Create(
    issuesPath,
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

  fmt.Println("Issue added successfully to the root tree of the first commit.")
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

func IssuesFromGitNotes(gitNotes []*git.Note) []*Issue {
	var uIssues []*Issue
	for _, notePtr := range gitNotes {
		note := *notePtr
		author := *note.Author()
		lines := strings.Split(note.Message(), "\n")

		for _, line := range lines {
			if line != "" {
				var uIssue Issue
				err := json.Unmarshal([]byte(line), &uIssue)
				if err != nil {
					fmt.Printf("Error unmarshaling JSON: %v", err)
					os.Exit(1)
				}

				uIssue.Author = author.Email
				// fmt.Printf("%+v\n", uIssue)
				uIssues = append(uIssues, &uIssue)
			}
		}
	}

	return uIssues
}
