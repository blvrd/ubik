package lens

import (
	// "encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/google/go-cmp/cmp"
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

		exampleName := parsedJSON.Search("name").Data().(string)
		shouldTestReverse := parsedJSON.Search("reverse").Data()
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
				lensSource := NewLensSource(lensJSON.Bytes())
				forwardResult := InterpretLens(originalPatch, lensSource)
				reverseResult := InterpretLens(forwardResult, lensSource.Reverse())
        fmt.Printf("forward result: %#v\n", forwardResult)
        fmt.Printf("reversed lens: %#v\n", lensSource.Reverse())
        fmt.Printf("reverse result: %#v\n", reverseResult)

				if diff := cmp.Diff(originalPatch, reverseResult); diff != "" {
					t.Errorf("ApplyLensToPatch mismatch (-want +got):\n%s", diff)
				}
			})
		}
	}
}
