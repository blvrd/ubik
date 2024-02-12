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
  "github.com/charmbracelet/log"
)

const (
	projectsPath = "refs/notes/ubik/projects"
	issuesPath   = "refs/notes/ubik/issues"
	commentsPath = "refs/notes/ubik/comments"
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
	Progress    int    `json:"progress"`
}

func (p Project) GetRefPath() string {
  return projectsPath
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
	Closed    string `json:"closed"`
	ParentType  string `json:"parent_type"`
	ParentId    string `json:"parent_id"`
}

func (i Issue) GetRefPath() string {
  return issuesPath
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
}

func (c Comment) GetRefPath() string {
  return commentsPath
}

func (c Comment) GetId() string {
  return c.Id
}

func (c Comment) Marshal() ([]byte, error) {
  return json.Marshal(c)
}

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
        project := Project{
          Id:          uuid.New().String(),
          Author:      GetAuthorEmail(), // Make sure you define this
          Title:       titleFlag,
          Description: descriptionFlag,
          Closed:      "false",
        }

        Add(project)
      }
		},
	}

	projectsAddCmd.Flags().StringP("title", "t", "", "Title for the project")
	projectsAddCmd.Flags().StringP("description", "d", "", "Description for the project")
	projectsAddCmd.Flags().Bool("termui", false, "Open the terminal UI")

  projectsAddCmd.MarkFlagsRequiredTogether("title", "description")

	var projectsUpdateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update",
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
      idFlag, _ := cmd.Flags().GetString("id")
      titleFlag, _ := cmd.Flags().GetString("title")
			descriptionFlag, _ := cmd.Flags().GetString("description")
      termUiFlag, _ := cmd.Flags().GetBool("termui")

      if termUiFlag {
        os.Exit(0)
      } else {
        project := Project{
          Id:          idFlag,
          Author:      GetAuthorEmail(), // Make sure you define this
          Title:       titleFlag,
          Description: descriptionFlag,
          Closed:      "false",
        }

        Update(project)
      }
		},
	}

	projectsUpdateCmd.Flags().String("id", "", "ID for the project")
	projectsUpdateCmd.Flags().String("title", "", "Title for the project")
	projectsUpdateCmd.Flags().String("description", "", "Description for the project")
	projectsUpdateCmd.Flags().Bool("termui", false, "Open the terminal UI")

  projectsUpdateCmd.MarkFlagsRequiredTogether("id", "title", "description")

  var projectsRemoveCmd = &cobra.Command{
		Use:   "remove",
		Short: "Remove",
		Run: func(cmd *cobra.Command, args []string) {
      idFlag, _ := cmd.Flags().GetString("id")

      entity := Project{
        Id: idFlag,
      }

      Remove(entity)
    },
	}

	projectsRemoveCmd.Flags().String("id", "", "ID for the project")

  projectsRemoveCmd.MarkFlagRequired("id")

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
      parentTypeFlag, _ := cmd.Flags().GetString("parent_type")

      issue := Issue{
        Id:          uuid.New().String(),
        Author:      GetAuthorEmail(), // Make sure you define this
        Title:       titleFlag,
        Description: descriptionFlag,
        Closed:      "false",
        ParentId:    parentIdFlag,
        ParentType:  parentTypeFlag,
      }

      Add(issue)
		},
	}

	issuesAddCmd.Flags().String("title", "", "Title for the issue")
	issuesAddCmd.Flags().String("description", "", "Description for the issue")
	issuesAddCmd.Flags().String("parent_id", "", "Parent ID for the issue")
	issuesAddCmd.Flags().String("parent_type", "", "Parent type for the issue")

  issuesAddCmd.MarkFlagRequired("title")
  issuesAddCmd.MarkFlagRequired("description")
  issuesAddCmd.MarkFlagRequired("parent_id")
  issuesAddCmd.MarkFlagRequired("parent_type")

	var commentsCmd = &cobra.Command{
		Use:   "comments",
		Short: "comments",
	}

	var commentsListCmd = &cobra.Command{
		Use:   "list",
		Short: "List comments you've created",
		Run: func(cmd *cobra.Command, args []string) { ListComments() },
	}

	var commentsAddCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a new thing",
		Run: func(cmd *cobra.Command, args []string) {
			descriptionFlag, _ := cmd.Flags().GetString("description")
      parentIdFlag, _ := cmd.Flags().GetString("parent_id")
      parentTypeFlag, _ := cmd.Flags().GetString("parent_type")

      comment := Comment{
        Id:          uuid.New().String(),
        Author:      GetAuthorEmail(),
        Description: descriptionFlag,
        ParentId:    parentIdFlag,
        ParentType:  parentTypeFlag,
      }

      Add(comment)
		},
	}

	commentsAddCmd.Flags().String("description", "", "Description for the comment")
	commentsAddCmd.Flags().String("parent_id", "", "Parent ID for the comment")
	commentsAddCmd.Flags().String("parent_type", "", "Parent type for the comment")

  commentsAddCmd.MarkFlagRequired("description")
  commentsAddCmd.MarkFlagRequired("parent_id")
  commentsAddCmd.MarkFlagRequired("parent_type")

  rootCmd.AddCommand(
    projectsCmd,
    issuesCmd,
    commentsCmd,
    termUiCmd,
    pushCmd,
    pullCmd,
    nukeCmd,
  )
  projectsCmd.AddCommand(projectsAddCmd, projectsUpdateCmd, projectsRemoveCmd, projectsListCmd)
  issuesCmd.AddCommand(issuesAddCmd, issuesListCmd)
  commentsCmd.AddCommand(commentsListCmd, commentsAddCmd)

	if err := rootCmd.Execute(); err != nil {
    log.Fatal(err)
	}
}

func Nuke() {
  exec.Command("./ubik_clear_all").Run()
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

func GetTree(commit *git.Commit) *git.Tree {
  // Getting the root tree of the first commit
  tree, err := commit.Tree()
  if err != nil {
    log.Fatalf("Failed to get root tree: %v\n", err)
  }

  return tree
}

func Add(entity Entity) error {
  wd := GetWd()
  repo, err := git.OpenRepository(wd)
  if err != nil {
    return fmt.Errorf("Failed to open repository: %v", err)
  }

  firstCommit := GetFirstCommit(repo)
  rootTree := GetTree(firstCommit)

  var newContent string
  note, err := repo.Notes.Read(entity.GetRefPath(), rootTree.Id())
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
    rootTree.Id(),
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
  rootTree := GetTree(firstCommit)

  var newContent string
  note, err := repo.Notes.Read(entity.GetRefPath(), rootTree.Id())
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
    rootTree.Id(),
    newContent,
    true,
    )
  if err != nil {
    return fmt.Errorf("Failed to add note to tree: %v", err)
  }

  return nil
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
  refPath := issuesPath
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

func ListComments() {
  refPath := commentsPath
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
  refPath := commentsPath
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

func GetWd() string {
  wd, err := os.Getwd()

  if err != nil {
		log.Fatalf("Failed to get current workding directory: %v", err)
  }

  return wd
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
		author := *note.Author()
		lines := strings.Split(note.Message(), "\n")

		for _, line := range lines {
			if line != "" {
				var uIssue Issue
				err := json.Unmarshal([]byte(line), &uIssue)
				if err != nil {
					log.Fatalf("Error unmarshaling JSON: %v", err)
				}

				uIssue.Author = author.Email
				// fmt.Printf("%+v\n", uIssue)
				uIssues = append(uIssues, &uIssue)
			}
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
