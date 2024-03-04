package entity

import (
	"bytes"
	"fmt"
	"github.com/charmbracelet/log"
	"os/exec"
	"strings"
)

const (
	ubikPrefix       = "refs/notes/ubik/*"
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

// DeleteRemoteRefs deletes all remote refs under a given namespace.
func (*GitRepository) DeleteRemoteRefs(remoteName, namespace string) error {
	// List all remote refs under the namespace
	cmd := exec.Command("git", "ls-remote", "--refs", remoteName, namespace+"*")
	var outList bytes.Buffer
	cmd.Stdout = &outList
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to list remote refs: %w", err)
	}

	// Parse the output to get ref names
	lines := strings.Split(outList.String(), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue // Skip invalid lines
		}
		ref := parts[1]

		// Construct the deletion refspec (e.g., ":refs/notes/ubik")
		deleteRefSpec := ":" + ref

		// Push the deletion refspec to the remote
		cmdDelete := exec.Command("git", "push", remoteName, deleteRefSpec)
		if err := cmdDelete.Run(); err != nil {
			return fmt.Errorf("failed to delete ref %s: %w", ref, err)
		}
		fmt.Printf("Successfully deleted ref %s\n", ref)
	}

	return nil
}

// DeleteLocalRefs deletes all local refs under the given namespace.
func (*GitRepository) DeleteLocalRefs(namespace string) error {
	// List all local refs under the namespace
	cmd := exec.Command("git", "for-each-ref", "--format=%(refname)", namespace+"*")
	var outList bytes.Buffer
	cmd.Stdout = &outList
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to list local refs: %w", err)
	}

	// Parse the output to get ref names
	lines := strings.Split(outList.String(), "\n")
	for _, ref := range lines {
		if ref == "" {
			continue // Skip empty lines
		}

		// Delete the local ref
		cmdDelete := exec.Command("git", "update-ref", "-d", ref)
		if err := cmdDelete.Run(); err != nil {
			return fmt.Errorf("failed to delete local ref %s: %w", ref, err)
		}
		fmt.Printf("Successfully deleted local ref %s\n", ref)
	}

	return nil
}
