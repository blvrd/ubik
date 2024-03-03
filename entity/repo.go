package entity

import (
  "fmt"
  "os/exec"
	"github.com/charmbracelet/log"
)

const (
  ubikPrefix = "refs/notes/ubik/*"
  ubikRemoteSuffix = "refs/notes/ubik/*"
)

type GitRepository struct {
  Path string
}

func NewGitRepository() *GitRepository {
  cwd := GetWd()
  return &GitRepository{Path: cwd}
}

func (*GitRepository) PushRefs(remote string) error {
  cmd := exec.Command("git", "push", remote, fmt.Sprintf("%s:%s", ubikPrefix, ubikRemoteSuffix))

  log.Infof("running cmd: %s", cmd.String())

  return cmd.Run()
}

func (*GitRepository) PullRefs(remote string) error {
  cmd := exec.Command("git", "fetch", remote, fmt.Sprintf("%s:%s", IssuesPath, IssuesPath))

  log.Infof("running cmd: %s", cmd.String())

  return cmd.Run()
}
