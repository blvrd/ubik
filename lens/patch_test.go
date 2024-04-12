package lens

import (
	// "encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/charmbracelet/log"
)

func findFocusedFiles(dir string) ([]string, error) {
	cmd := exec.Command("grep", "-rl", `"focus": true`, dir)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// grep returns exit code 1 if no matches are found
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
		t.Errorf("Error finding files: %v", err)
	}

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			t.Errorf("Error opening file: %v", err)
		}
		defer f.Close()

		parsedJSON, err := gabs.ParseJSONFile(f.Name())
		if err != nil {
			// t.Fatalf("Could not parse JSON file: %s", f.Name())
			t.Fatalf("Could not parse JSON file: %v", err)
		}

		exampleName := parsedJSON.Search("name").Data().(string)
		originalDocJSON := parsedJSON.Search("original")
		evolvedDocJSON := parsedJSON.Search("evolved")
		lensJSON := parsedJSON.Search("lens")

		t.Run(fmt.Sprintf("%s (forward)", exampleName), func(t *testing.T) {
			lensSource := NewLensSource(lensJSON.Bytes())
			result := ApplyLensToDoc(lensSource, originalDocJSON)

			if diff := cmp.Diff(evolvedDocJSON, result, cmp.AllowUnexported(gabs.Container{})); diff != "" {
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
    fmt.Println("Error:", err)
    os.Exit(1)
  }

  if len(focusedFiles) > 0 {
    files = append(files, focusedFiles...)
  } else {
    testFiles, err := findTestFiles(testDir)
    if err != nil {
      fmt.Println("Error:", err)
      os.Exit(1)
    }
    files = append(files, testFiles...)
  }

	for _, file := range files {

		f, err := os.Open(file)
		if err != nil {
			t.Errorf("Error opening file: %v", err)
		}
		defer f.Close()

		parsedJSON, err := gabs.ParseJSONFile(f.Name())
		if err != nil {
			t.Fatalf("Could not parse JSON file: %s", f.Name())
		}

		exampleName := parsedJSON.Search("name").Data().(string)
		shouldTestReverse := parsedJSON.Search("reversed")
		originalPatchJSON := parsedJSON.Search("original")
		evolvedPatchJSON := parsedJSON.Search("evolved")
		lensJSON := parsedJSON.Search("lens")

		t.Run(fmt.Sprintf("%s forward", exampleName), func(t *testing.T) {
			originalPatch := NewPatchFromJSON(originalPatchJSON.Bytes())
			evolvedPatch := NewPatchFromJSON(evolvedPatchJSON.Bytes())
			lensSource := NewLensSource(lensJSON.Bytes())
			result := InterpretLens(originalPatch, lensSource)

			if diff := cmp.Diff(evolvedPatch, result); diff != "" {
				t.Errorf("ApplyLensToPatch mismatch (-want +got):\n%s", diff)
			}
		})

		if shouldTestReverse != nil {
			t.Run(fmt.Sprintf("%s reverse", exampleName), func(t *testing.T) {
				originalPatch := NewPatchFromJSON(originalPatchJSON.Bytes())
				expectedReversedPatch := NewPatchFromJSON(shouldTestReverse.Bytes())
				lensSource := NewLensSource(lensJSON.Bytes())
				forwardResult := InterpretLens(originalPatch, lensSource)
				reverseResult := InterpretLens(forwardResult, lensSource.Reverse())

				if diff := cmp.Diff(expectedReversedPatch, reverseResult); diff != "" {
					t.Errorf("ApplyLensToPatch mismatch (-want +got):\n%s", diff)
				}
			})
		}
	}
}
