package lens

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func findFocusedFiles(dir string) ([]string, error) {
	cmd := exec.Command("grep", "-rl", `"focus": true`, dir)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, err
	}

	filePaths := strings.Split(strings.TrimSpace(string(output)), "\n")
	return filePaths, nil
}

func findTestFiles(dir string) ([]string, error) {
	return filepath.Glob(filepath.Join(dir, "*.json"))
}

func TestLensDoc(t *testing.T) {
	log.SetLevel(log.InfoLevel)
	pattern := "./doc_tests/*_doc_test.json"

	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Error finding files: %v", err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("Error reading file: %v", err)
			return
		}

		type testConfig struct {
			ExampleName string          `json:"name"`
			OriginalDoc json.RawMessage `json:"original"`
			EvolvedDoc  json.RawMessage `json:"evolved"`
			Lens        json.RawMessage `json:"lens"`
		}

		var config testConfig
		err = json.Unmarshal(data, &config)

		exampleName := config.ExampleName
		originalDocJSON := config.OriginalDoc
		evolvedDocJSON := config.EvolvedDoc

		lensJSON := config.Lens

		t.Run(fmt.Sprintf("%s (forward)", exampleName), func(t *testing.T) {
			lensSource := NewLensSource(lensJSON)
			result := ApplyLensToDoc(lensSource, originalDocJSON)

			var evolvedDocBuffer bytes.Buffer
			err := json.Compact(&evolvedDocBuffer, evolvedDocJSON)
			if err != nil {
				t.Fatalf("Failed to normalize evolved JSON: %v", err)
			}
			normalizedEvolvedDocJSON := evolvedDocBuffer.Bytes()

			var resultBuffer bytes.Buffer
			err = json.Compact(&resultBuffer, result)
			if err != nil {
				t.Fatalf("Failed to normalize result JSON: %v", err)
			}

			normalizedResultJSON := resultBuffer.Bytes()

			normalizeJSON := func(x []byte) map[string]interface{} {
				var m map[string]interface{}
				json.Unmarshal(x, &m)
				return m
			}

			if diff := cmp.Diff(normalizedEvolvedDocJSON, normalizedResultJSON, cmpopts.AcyclicTransformer("normalizeJSON", normalizeJSON)); diff != "" {
				t.Errorf("ApplyLensToDoc mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLensPatch(t *testing.T) {
	var files []string
	testDir := "./patch_tests"

	focusedFiles, err := findFocusedFiles(testDir)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	if len(focusedFiles) > 0 {
		files = append(files, focusedFiles...)
	} else {
		testFiles, err := findTestFiles(testDir)
		if err != nil {
			t.Fatalf("error: %v\n", err)
		}
		files = append(files, testFiles...)
	}

	for _, file := range files {

		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("error: %v\n", err)
		}

		type testConfig struct {
			ExampleName   string          `json:"name"`
			OriginalPatch json.RawMessage `json:"original"`
			EvolvedPatch  json.RawMessage `json:"evolved"`
			Lens          json.RawMessage `json:"lens"`
			Reverse       json.RawMessage `json:"reverse"`
		}

		var config testConfig
		err = json.Unmarshal(data, &config)

		exampleName := config.ExampleName
		shouldTestReverse := config.Reverse
		originalPatchJSON := config.OriginalPatch
		evolvedPatchJSON := config.EvolvedPatch
		lensJSON := config.Lens

		t.Run(fmt.Sprintf("%s forward", exampleName), func(t *testing.T) {
			originalPatch := NewPatchFromJSON(originalPatchJSON)
			evolvedPatch := NewPatchFromJSON(evolvedPatchJSON)
			lensSource := NewLensSource(lensJSON)
			result := InterpretLens(originalPatch, lensSource)

			if diff := cmp.Diff(evolvedPatch, result); diff != "" {
				t.Errorf("ApplyLensToPatch mismatch (-want +got):\n%s", diff)
			}
		})

		if shouldTestReverse != nil {
			t.Run(fmt.Sprintf("%s reverse", exampleName), func(t *testing.T) {
				originalPatch := NewPatchFromJSON(originalPatchJSON)
				expectedReversedPatch := NewPatchFromJSON(shouldTestReverse)
				lensSource := NewLensSource(lensJSON)
				forwardResult := InterpretLens(originalPatch, lensSource)
				reverseResult := InterpretLens(forwardResult, lensSource.Reverse())

				if diff := cmp.Diff(expectedReversedPatch, reverseResult); diff != "" {
					t.Errorf("ApplyLensToPatch mismatch (-want +got):\n%s", diff)
				}
			})
		}
	}
}
