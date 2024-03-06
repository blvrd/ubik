package entity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
  "time"

	"github.com/charmbracelet/log"
)

type Note struct {
  ObjectId string
  AttachedObjectId string
  Ref string
  Stale bool
}

func (n *Note) Read() (map[string]interface{}, error) {
  var content map[string]interface{}
	cmd := exec.Command("git", "cat-file", "-p", n.ObjectId)

	var output bytes.Buffer
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list local refs: %w", err)
	}

  json.Unmarshal(output.Bytes(), &content)

  return content, nil
}

func (n *Note) Merge(otherNote *Note) error {
  map1, err := n.Read()
  if err != nil {
    return err
  }

  map2, err := otherNote.Read()
  if err != nil {
    return err
  }

  mergeMaps(map1, map2)

  noteJson, err := json.Marshal(map1)

	cmd := exec.Command("git", "notes", "--ref", n.Ref, "add", "-f", "-m", string(noteJson), n.AttachedObjectId)
  log.Infof("running cmd: %s", cmd.String())
  output, err := cmd.CombinedOutput()
  log.Infof("Command output: %s\n", output)

  if err != nil {
    return fmt.Errorf("Command execution failed: %v\n", err)
  }

  n.Stale = true
  return nil
}

func parseTime(timeStr string) (time.Time, bool) {
  const layout = "2006-01-02T15:04:05Z" // ISO 8601 format
  t, err := time.Parse(layout, timeStr)
  return t, err == nil
}

func shouldUpdate(value1, value2 map[string]interface{}) bool {
  updatedAt1Str, ok1 := value1["updated_at"].(string)
  updatedAt2Str, ok2 := value2["updated_at"].(string)
  if !ok1 || !ok2 {
    return false
  }
  updatedAt1, ok1 := parseTime(updatedAt1Str)
  updatedAt2, ok2 := parseTime(updatedAt2Str)
  if !ok1 || !ok2 {
    return false
  }

  if updatedAt2.After(updatedAt1) {
    log.Infof("Should update %s with values from %s", value1, value2)
  }

  return updatedAt2.After(updatedAt1)
}

func mergeMaps(map1, map2 map[string]interface{}) {
  for key, value2 := range map2 {
    value2Map, ok2 := value2.(map[string]interface{})
    if !ok2 {
      continue
    }
    value1, exists := map1[key]
    if !exists {
      log.Infof("Adding new entry from remote note")
      map1[key] = value2 // Add new entry
      continue
    }
    value1Map, ok1 := value1.(map[string]interface{})
    if !ok1 {
      continue
    }
    if deletedAt2, exists := value2Map["deleted_at"].(string); exists {
      // Update deleted_at in map1 to match map2
      value1Map["deleted_at"] = deletedAt2
      map1[key] = value1Map
      continue
    }
    if shouldUpdate(value1Map, value2Map) {
      map1[key] = value2 // Update existing entry
    }
  }
}
