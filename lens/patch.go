package lens

import (
	"encoding/json"
	"strconv"
	"strings"


	"github.com/Jeffail/gabs/v2"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/wI2L/jsondiff"
	// "github.com/charmbracelet/log"
)

func PatchToDoc(p Patch) *gabs.Container {
	var original []byte
	var filtered Patch

	for _, patchOp := range p {
		if patchOp.Op != "noop" {
			filtered = append(filtered, patchOp)
		}
	}

	patchJSON, err := json.Marshal(filtered)
	if err != nil {
		panic(err)
	}
	var doc *gabs.Container
	original = []byte(`{}`)
	patch, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		panic(err)
	}

	modified, err := patch.Apply(original)
	if err != nil {
		panic(err)
	}

	doc, err = gabs.ParseJSON(modified)
	if err != nil {
		panic(err)
	}

	return doc
}

func ApplyLensToDoc(ls LensSource, doc *gabs.Container) *gabs.Container {
	// Let's create a merge patch from these two documents...
	// original := []byte(`{"name": "John", "age": 24, "height": 3.21}`)
	// original := []byte(`{"name": "Jane", "age": 24}`)
	original := doc.Bytes()
	patch := DocToPatch(original)
	evolvedPatch := InterpretLens(patch, ls)
	x := PatchToDoc(evolvedPatch)
	return x
}

func DocToPatch(target []byte) Patch {
	empty := []byte(`{}`)

	p, err := jsondiff.CompareJSON(empty, target)
	if err != nil {
		panic(err)
	}
	patchJSON, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	// parsedJSON, err := gabs.ParseJSON(patchJSON)
	// if err != nil {
	// 	panic(err)
	// }

	return NewPatchFromJSON(patchJSON)
}

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type Patch []PatchOperation

func InterpretLens(patches []PatchOperation, lenses []Lens) []PatchOperation {
	var result []PatchOperation

	for _, patch := range patches {
		transformedPatch := applyLens(patch, lenses)
		if transformedPatch.Path != "" || transformedPatch.Value != nil {
			result = append(result, transformedPatch)
		}
	}

	return result

}

func applyLens(patch PatchOperation, lenses []Lens) PatchOperation {
	for _, lens := range lenses {
		if lens.Rename != nil {
			// Perform rename operation
			if strings.HasSuffix(patch.Path, "/"+lens.Rename.Source) {
				patch.Path = strings.TrimSuffix(patch.Path, "/"+lens.Rename.Source) + "/" + lens.Rename.Destination
			}
		} else if lens.Convert != nil {
			// Perform convert operation
			if strings.HasSuffix(patch.Path, "/"+lens.Convert.Name) {
				if value, ok := patch.Value.(string); ok {
					for _, mapping := range lens.Convert.Mapping {
						for k, v := range mapping {
							if value == k {
								patch.Value = v
								break
							}
						}
					}
				}
			}
		} else if lens.Head != nil {
			// Perform head operation
			if strings.HasSuffix(patch.Path, "/"+lens.Head.Name) {
				if slice, ok := patch.Value.([]interface{}); ok && len(slice) > 0 {
					patch.Value = slice[0]
				}
			}
		} else if lens.In != nil {
			if strings.HasPrefix(patch.Path, "/"+lens.In.Name) {
				if arr, ok := patch.Value.([]interface{}); ok {
					for i := range arr {
						if obj, ok := arr[i].(map[string]interface{}); ok {
              jsonBytes, err := json.Marshal(obj)
              if err != nil {
                panic(err)
              }
							nestedPatches := DocToPatch(jsonBytes)
							for _, nestedPatch := range nestedPatches {
								nestedPatch.Path = patch.Path + "/" + strconv.Itoa(i) + nestedPatch.Path
								for _, nestedLens := range lens.In.Lens {
									nestedPatch = applyLens(nestedPatch, []Lens{nestedLens})
								}
							}
							arr[i] = PatchToDoc(nestedPatches)
						}
					}
					patch.Value = arr
				} else if obj, ok := patch.Value.(map[string]interface{}); ok {
          jsonBytes, err := json.Marshal(obj)
          if err != nil {
            panic(err)
          }
					nestedPatches := DocToPatch(jsonBytes)
          var appliedNestedPatches []PatchOperation
					for _, nestedPatch := range nestedPatches {
						for _, nestedLens := range lens.In.Lens {
							nestedPatch = applyLens(nestedPatch, []Lens{nestedLens})
              appliedNestedPatches = append(appliedNestedPatches, nestedPatch)
						}
					}
					patch.Value = PatchToDoc(appliedNestedPatches).Data()
				}
			}
		} else if lens.Hoist != nil {
			// Perform hoist operation
			if strings.HasPrefix(patch.Path, "/"+lens.Hoist.Host) {
				if nestedData, ok := patch.Value.(map[string]interface{}); ok {
					if hoistValue, ok := nestedData[lens.Hoist.Name]; ok {
						patch.Path = strings.TrimPrefix(patch.Path, "/"+lens.Hoist.Host) + "/" + lens.Hoist.Name
						patch.Value = hoistValue
					}
				}
			}
		} else if lens.Remove != nil {
			// Perform remove operation
			if strings.HasSuffix(patch.Path, "/"+lens.Remove.Name) {
				return PatchOperation{} // Return an empty patch to remove the field
			}
		}
	}
	return patch
}

// func (p *PatchOperation) MarshalJSON() ([]byte, error) {
// 	b, err := json.Marshal(p.JSON)
//
// 	return b, err
// }

// func NewPatchOperationFromJSON(c *gabs.Container) PatchOperation {
// 	return PatchOperation{JSON: c}
// }

func NewPatchFromJSON(jsonData []byte) Patch {
	var patch Patch

	err := json.Unmarshal(jsonData, &patch)
	if err != nil {
		panic(err)
	}

	return patch
}
