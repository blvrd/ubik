package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/blvrd/ubik/entity"
	"github.com/blvrd/ubik/ui"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
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
	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(true)

	// ========================
	// CLI Commands
	// ========================

	// rootCmd represents the base command when called without any subcommands
	var rootCmd = &cobra.Command{
		Use:   "ubik",
		Short: "Use Ubik from the handy Terminal UI",
		Run: func(cmd *cobra.Command, args []string) {
			if err := ui.Run(); err != nil {
				log.Fatal(err)
			}
		},
	}

	var pushCmd = &cobra.Command{
		Use:   "push",
		Short: "push",
		Run: func(cmd *cobra.Command, args []string) {
			repo := entity.NewGitRepository()
			err := repo.PushRefs("origin")
			if err != nil {
				log.Fatalf("error pushing refs: %v", err)
			}
		},
	}

	var pullCmd = &cobra.Command{
		Use:   "pull",
		Short: "pull",
		Run: func(cmd *cobra.Command, args []string) {
			repo := entity.NewGitRepository()
			err := repo.PullRefs("origin")
			if err != nil {
				log.Fatalf("error pulling refs: %v", err)
			}

			err = repo.MergeRefs()
			if err != nil {
				log.Fatalf("error merging refs: %v", err)
			}

			err = repo.DeleteLocalRefs("refs/notes/ubik/merging/issues")
			if err != nil {
				log.Fatalf("error merging refs: %v", err)
			}
		},
	}

	// var issuesCmd = &cobra.Command{
	// 	Use:   "issues",
	// 	Short: "issues",
	// }
	//
	// var issuesListCmd = &cobra.Command{
	// 	Use:   "list",
	// 	Short: "List issues you've created",
	// 	Run:   func(cmd *cobra.Command, args []string) { entity.ListIssues() },
	// }
	//
	// var issuesAddCmd = &cobra.Command{
	// 	Use:   "add",
	// 	Short: "Add a new thing",
	// 	Run: func(cmd *cobra.Command, args []string) {
	// 		titleFlag, _ := cmd.Flags().GetString("title")
	// 		descriptionFlag, _ := cmd.Flags().GetString("description")
	// 		parentIdFlag, _ := cmd.Flags().GetString("parent_id")
	// 		parentTypeFlag, _ := cmd.Flags().GetString("parent_type")
	//
	// 		issue := entity.Issue{
	// 			Id:          uuid.New().String(),
	// 			Author:      entity.GetAuthorEmail(), // Make sure you define this
	// 			Title:       titleFlag,
	// 			Description: descriptionFlag,
	// 			ClosedAt:    time.Time{},
	// 			ParentId:    parentIdFlag,
	// 			ParentType:  parentTypeFlag,
	// 			RefPath:     entity.IssuesPath,
	// 			CreatedAt:   time.Now().UTC(),
	// 			UpdatedAt:   time.Now().UTC(),
	// 		}
	//
	// 		entity.Add(&issue)
	// 	},
	// }
	//
	// issuesAddCmd.Flags().String("title", "", "Title for the issue")
	// issuesAddCmd.Flags().String("description", "", "Description for the issue")
	// issuesAddCmd.Flags().String("parent_id", "", "Parent ID for the issue")
	// issuesAddCmd.Flags().String("parent_type", "", "Parent type for the issue")
	//
	// issuesAddCmd.MarkFlagsRequiredTogether("title", "description")
	// issuesAddCmd.MarkFlagsRequiredTogether("parent_id", "parent_type")
	//

	rootCmd.AddCommand(
		pushCmd,
		pullCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func Nuke() {
	exec.Command("./ubik_clear_all").Run()
	repo := entity.NewGitRepository()

	remoteName := "origin"
	namespace := "refs/notes/ubik"

	if err := repo.DeleteLocalRefs(namespace); err != nil {
		fmt.Println("Error deleting local refs:", err)
	} else {
		fmt.Println("Local refs deleted successfully.")
	}

	if err := repo.DeleteRemoteRefs(remoteName, namespace); err != nil {
		fmt.Println("Error deleting remote refs:", err)
	} else {
		fmt.Println("Remote refs deleted successfully.")
	}
}
