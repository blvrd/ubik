package entity

import (
	"bytes"
	"fmt"
	// "os"
	"os/exec"
	"strings"
  "encoding/json"

	"github.com/charmbracelet/log"
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

func (*GitRepository) Notes(ref string) ([]Note, error) {
  var notes []Note
	cmd := exec.Command("git", "notes", "--ref", ref, "list")

	log.Infof("running cmd: %s", cmd.String())

	var output bytes.Buffer
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list local refs: %w", err)
	}

	lines := strings.Split(output.String(), "\n")
  for _, line := range lines {
    if line == "" {
      continue // Skip empty lines
    }
    parts := strings.Split(line, " ")
    if len(parts) != 2 {
      continue // Skip lines that don't have exactly two parts
    }
    note := Note{
      ObjectId:         parts[0],
      AttachedObjectId: parts[1],
      Ref:              ref,
    }
    notes = append(notes, note)
  }

  return notes, nil
}

func (*GitRepository) PushRefs(remote string) error {
	cmd := exec.Command("git", "push", remote, fmt.Sprintf("%s:%s", ubikPrefix, ubikRemoteSuffix))

	log.Infof("running cmd: %s", cmd.String())

	return cmd.Run()
}

func (*GitRepository) PullRefs(remote string) error {
	cmd := exec.Command("git", "fetch", remote, fmt.Sprintf("%s:%s", IssuesPath, "refs/notes/ubik/merging/issues"))

	log.Infof("running cmd: %s", cmd.String())

	return cmd.Run()
}

func (r *GitRepository) MergeRefs() error {
  remoteNamespace := "refs/notes/ubik/merging/issues"
  localNamespace := "refs/notes/ubik/issues"

  localNotes, err := r.Notes(localNamespace)
  if err != nil {
    return err
  }
  remoteNotes, err := r.Notes(remoteNamespace)
  if err != nil {
    return err
  }

  for _, remoteNote := range remoteNotes {
    log.Infof("Remote note: %+v", remoteNote)
    if len(localNotes) == 0 {
      remoteNoteMap, err := remoteNote.Read()
      if err != nil {
        return nil
      }

      remoteNoteJSON, err := json.Marshal(remoteNoteMap)
	    cmd := exec.Command("git", "notes", "--ref", localNamespace, "add", "-f", "-m", string(remoteNoteJSON), remoteNote.AttachedObjectId)
      log.Infof("running cmd: %s", cmd.String())
      output, err := cmd.CombinedOutput()
      log.Infof("Command output: %s\n", output)
      if err != nil {
        return fmt.Errorf("Command execution failed: %v\n", err)
      }
    }

    for _, localNote := range localNotes {
      log.Infof("Local note: %+v", localNote)
      if remoteNote.AttachedObjectId == localNote.AttachedObjectId {
        log.Info("merging")
        localNote.Merge(&remoteNote)
      }
    }
  }
  return nil
  // var err error
	//  remoteNamespace := "refs/notes/ubik/merging/issues"
	//  localNamespace := "refs/notes/ubik/issues"
	//
	//  localNotes := r.Notes(localNamespace)
	//  remoteNotes := r.Notes(remoteNamespace)
	//
	//  // first argument is the target, second is collection to merge in from
	//  // build a map by git object id that the note is attached to, and make sure ref paths match first:
	//  // {
	//  //   "asdf12345": { to: "noteid", from: note }
	//  // }
	//  // or maybe it just loops through each list and finds the matching object in the other one
	//  // should return collection of local notes to save
	//  r.MergeNotes(localNotes, remoteNotes)
	//
	//  // get the note for each namespace
	//  // parse JSON for each
	//  // merge JSON
	//  // write back to local namespace with new merged JSON
	//  // delete refs in remote namespace
	//
	// cmd := exec.Command("git", "for-each-ref", "--format=%(refname)", remoteNamespace)
	// var output bytes.Buffer
	// cmd.Stdout = &output
	// if err := cmd.Run(); err != nil {
	// 	return fmt.Errorf("failed to list local refs: %w", err)
	// }
	//
	// // Parse the output to get ref names
	// lines := strings.Split(output.String(), "\n")
	// for _, ref := range lines {
	// 	if ref == "" {
	// 		continue // Skip empty lines
	// 	}
	//
	// 	cmdRemoteNotesList := exec.Command("git", "notes", "--ref", ref, "list")
	//    cmdRemoteNotesList.Stdout = os.Stdout
	// 	if err := cmdRemoteNotesList.Run(); err != nil {
	// 		return fmt.Errorf("failed to read local ref %s: %w", ref, err)
	// 	}
	// }
	//
	//  return err
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
