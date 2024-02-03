package main

import (
	"fmt"
  "os"
  "os/exec"
)

type note struct {
  Id string `json:"id"`
  Author string `json:"author"`
  Content string `json:"content"`
}

func main() {
  configAuthor, err := exec.Command("git", "config", "user.email").Output()

  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }

	author := os.Getenv("GIT_AUTHOR_EMAIL")

	if author == "" {
		author = string(configAuthor)
	}

  refPath := fmt.Sprintf("refs/notes/ubik/notes/%s", author)

  notesJsonL, err := exec.Command("git", "notes", "--ref", refpath, )
  fmt.Println("")
  fmt.Printf(refPath)
  fmt.Println("")
}
