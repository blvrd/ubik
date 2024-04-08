package lens

import (
	// "encoding/json"
	"github.com/google/go-cmp/cmp"
	"os"
	"path/filepath"
	"testing"
  "github.com/Jeffail/gabs/v2"
)
func TestLensDoc(t *testing.T) {
	pattern := "*_doc_test.json"

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

    originalDocJSON := parsedJSON.Search("original")
    evolvedDocJSON := parsedJSON.Search("evolved")
    lensJSON := parsedJSON.Search("lens")

		t.Run("forward", func(t *testing.T) {
      lensSource := NewLensSource(lensJSON)
			result := ApplyLensToDoc(lensSource, originalDocJSON)

			if diff := cmp.Diff(evolvedDocJSON, result, cmp.AllowUnexported(gabs.Container{})); diff != "" {
				t.Errorf("ApplyLensToDoc mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLensPatch(t *testing.T) {
	pattern := "*_patch_test.json"

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
      t.Fatalf("Could not parse JSON file: %s", f.Name())
    }

    originalPatchJSON := parsedJSON.Search("original")
    evolvedPatchJSON := parsedJSON.Search("evolved")
    lensJSON := parsedJSON.Search("lens")

		t.Run("forward", func(t *testing.T) {
      originalPatch := NewPatch(originalPatchJSON)
      evolvedPatch := NewPatch(evolvedPatchJSON)
      lensSource := NewLensSource(lensJSON)
			result := ApplyLensToPatch(lensSource, originalPatch)

			if diff := cmp.Diff(evolvedPatch, result, cmp.AllowUnexported(gabs.Container{})); diff != "" {
				t.Errorf("ApplyLensToPatch mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
