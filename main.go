package main

import (
	"fmt"
	"os"
	"os/exec"
  "time"

	"github.com/spf13/cobra"
  "github.com/google/uuid"
  "github.com/charmbracelet/log"
  "github.com/blvrd/ubik/tui"
  "github.com/blvrd/ubik/entity"
)

func main() {
  // Open or create the log file
  logFile, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
  if err != nil {
    panic("could not open log file")
  }
  defer logFile.Close()

  // Set the global logger output to the file
  log.SetOutput(logFile)

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
    Run: func(cmd *cobra.Command, args []string) {
      if err := tui.Run(); err != nil {
        log.Fatal(err)
      }
    },
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
		Run: func(cmd *cobra.Command, args []string) { entity.ListProjects() },
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
        project := entity.Project{
          Id:          uuid.New().String(),
          Author:      entity.GetAuthorEmail(), // Make sure you define this
          Title:       titleFlag,
          Description: descriptionFlag,
          Closed:      "false",
          RefPath:     entity.ProjectsPath,
          CreatedAt:   time.Now().UTC(),
          UpdatedAt:   time.Now().UTC(),
        }

        entity.Add(project)
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
        project := entity.Project{
          Id:          idFlag,
          Author:      entity.GetAuthorEmail(), // Make sure you define this
          Title:       titleFlag,
          Description: descriptionFlag,
          Closed:      "false",
          RefPath:     entity.ProjectsPath,
          CreatedAt:   time.Now().UTC(), // TODO fix this - we need to find the record first
          UpdatedAt:   time.Now().UTC(),
        }

        entity.Update(project)
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

      e := entity.Project{
        Id: idFlag,
      }

      entity.Remove(e)
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
		Run: func(cmd *cobra.Command, args []string) { entity.ListIssues() },
	}

	var issuesAddCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a new thing",
		Run: func(cmd *cobra.Command, args []string) {
      titleFlag, _ := cmd.Flags().GetString("title")
			descriptionFlag, _ := cmd.Flags().GetString("description")
      parentIdFlag, _ := cmd.Flags().GetString("parent_id")
      parentTypeFlag, _ := cmd.Flags().GetString("parent_type")

      issue := entity.Issue{
        Id:          uuid.New().String(),
        Author:      entity.GetAuthorEmail(), // Make sure you define this
        Title:       titleFlag,
        Description: descriptionFlag,
        Closed:      "false",
        ParentId:    parentIdFlag,
        ParentType:  parentTypeFlag,
        RefPath:     entity.IssuesPath,
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
      }

      entity.Add(issue)
		},
	}

	issuesAddCmd.Flags().String("title", "", "Title for the issue")
	issuesAddCmd.Flags().String("description", "", "Description for the issue")
	issuesAddCmd.Flags().String("parent_id", "", "Parent ID for the issue")
	issuesAddCmd.Flags().String("parent_type", "", "Parent type for the issue")

  issuesAddCmd.MarkFlagsRequiredTogether("title", "description")
  issuesAddCmd.MarkFlagsRequiredTogether("parent_id", "parent_type")

	var issuesUpdateCmd = &cobra.Command{
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
      parentIdFlag, _ := cmd.Flags().GetString("parent_id")
      parentTypeFlag, _ := cmd.Flags().GetString("parent_type")
      termUiFlag, _ := cmd.Flags().GetBool("termui")

      if termUiFlag {
        os.Exit(0)
      } else {
        issue := entity.Issue{
          Id:          idFlag,
          Author:      entity.GetAuthorEmail(), // Make sure you define this
          Title:       titleFlag,
          Description: descriptionFlag,
          Closed:      "false",
          ParentId:    parentIdFlag,
          ParentType:  parentTypeFlag,
          RefPath:     entity.IssuesPath,
          CreatedAt:   time.Now().UTC(), // TODO fix this - we need to find the record first
          UpdatedAt:   time.Now().UTC(),
        }

        entity.Update(issue)
      }
		},
	}

	issuesUpdateCmd.Flags().String("id", "", "ID for the issue")
	issuesUpdateCmd.Flags().String("title", "", "Title for the issue")
	issuesUpdateCmd.Flags().String("description", "", "Description for the issue")
	issuesUpdateCmd.Flags().String("parent_id", "", "Parent ID for the issue")
	issuesUpdateCmd.Flags().String("parent_type", "", "Parent type for the issue")

  issuesUpdateCmd.MarkFlagsRequiredTogether("id", "title", "description")
  issuesUpdateCmd.MarkFlagsRequiredTogether("parent_id", "parent_type")

  var issuesRemoveCmd = &cobra.Command{
		Use:   "remove",
		Short: "Remove",
		Run: func(cmd *cobra.Command, args []string) {
      idFlag, _ := cmd.Flags().GetString("id")

      e := entity.Issue{
        Id: idFlag,
      }

      entity.Remove(e)
    },
	}

	issuesRemoveCmd.Flags().String("id", "", "ID for the issue")

  issuesRemoveCmd.MarkFlagRequired("id")

	var commentsCmd = &cobra.Command{
		Use:   "comments",
		Short: "comments",
	}

	var commentsListCmd = &cobra.Command{
		Use:   "list",
		Short: "List comments you've created",
		Run: func(cmd *cobra.Command, args []string) { entity.ListComments() },
	}

	var commentsAddCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a new thing",
		Run: func(cmd *cobra.Command, args []string) {
			descriptionFlag, _ := cmd.Flags().GetString("description")
      parentIdFlag, _ := cmd.Flags().GetString("parent_id")
      parentTypeFlag, _ := cmd.Flags().GetString("parent_type")

      comment := entity.Comment{
        Id:          uuid.New().String(),
        Author:      entity.GetAuthorEmail(),
        Description: descriptionFlag,
        ParentId:    parentIdFlag,
        ParentType:  parentTypeFlag,
        RefPath:     entity.CommentsPath,
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
      }

      entity.Add(comment)
		},
	}

	commentsAddCmd.Flags().String("description", "", "Description for the comment")
	commentsAddCmd.Flags().String("parent_id", "", "Parent ID for the comment")
	commentsAddCmd.Flags().String("parent_type", "", "Parent type for the comment")

  commentsAddCmd.MarkFlagRequired("description")
  commentsAddCmd.MarkFlagRequired("parent_id")
  commentsAddCmd.MarkFlagRequired("parent_type")

	var commentsUpdateCmd = &cobra.Command{
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
			descriptionFlag, _ := cmd.Flags().GetString("description")
      parentIdFlag, _ := cmd.Flags().GetString("parent_id")
      parentTypeFlag, _ := cmd.Flags().GetString("parent_type")
      termUiFlag, _ := cmd.Flags().GetBool("termui")

      if termUiFlag {
        os.Exit(0)
      } else {
        comment := entity.Comment{
          Id:          idFlag,
          Author:      entity.GetAuthorEmail(), // Make sure you define this
          Description: descriptionFlag,
          ParentId:    parentIdFlag,
          ParentType:  parentTypeFlag,
          RefPath:     entity.CommentsPath,
          CreatedAt:   time.Now().UTC(), // TODO fix this - we need to find the record first
          UpdatedAt:   time.Now().UTC(),
        }

        entity.Update(comment)
      }
		},
	}

	commentsUpdateCmd.Flags().String("id", "", "ID for the comment")
	commentsUpdateCmd.Flags().String("description", "", "Description for the comment")
	commentsUpdateCmd.Flags().String("parent_id", "", "Parent ID for the comment")
	commentsUpdateCmd.Flags().String("parent_type", "", "Parent type for the comment")

  commentsAddCmd.MarkFlagRequired("id")
  commentsAddCmd.MarkFlagRequired("description")
  commentsAddCmd.MarkFlagRequired("parent_id")
  commentsAddCmd.MarkFlagRequired("parent_type")

  var commentsRemoveCmd = &cobra.Command{
		Use:   "remove",
		Short: "Remove",
		Run: func(cmd *cobra.Command, args []string) {
      idFlag, _ := cmd.Flags().GetString("id")

      e := entity.Comment{
        Id: idFlag,
      }

      entity.Remove(e)
    },
	}

	commentsRemoveCmd.Flags().String("id", "", "ID for the comment")

  commentsRemoveCmd.MarkFlagRequired("id")

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
  issuesCmd.AddCommand(issuesAddCmd, issuesUpdateCmd, issuesRemoveCmd, issuesListCmd)
  commentsCmd.AddCommand(commentsAddCmd, commentsUpdateCmd, commentsRemoveCmd, commentsListCmd)

	if err := rootCmd.Execute(); err != nil {
    log.Fatal(err)
	}
}

func Nuke() {
  exec.Command("./ubik_clear_all").Run()
}

