package lens

import (
	"encoding/json"
	"strconv"
	"strings"
  // "fmt"

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
	Op         string      `json:"op"`
	Path       string      `json:"path"`
	Value      interface{} `json:"value,omitempty"`
  LensSource *LensSource `json:"lens,omitempty"`
}

type Patch []PatchOperation

func InterpretLens(patches Patch, lenses []Lens) Patch {
	var result Patch

	for _, lens := range lenses {
		var transformedPatches []PatchOperation

		for _, patch := range patches {
			transformedPatch := applyLens(patch, lens)
			if transformedPatch.Op != "" {
				transformedPatches = append(transformedPatches, transformedPatch)
			}
		}

		if lens.Add != nil {
			addPatch := PatchOperation{
				Op:    "add",
				Path:  "/" + lens.Add.Name,
				Value: lens.Add.Default,
			}
			transformedPatches = append(transformedPatches, addPatch)
		}

		patches = transformedPatches
	}

	result = patches

	if result == nil {
		return Patch{}
	}
	return result
}

func applyLens(patchOp PatchOperation, lens Lens) PatchOperation {
	if lens.Rename != nil {
		if strings.HasSuffix(patchOp.Path, "/"+lens.Rename.Source) {
			patchOp.Path = strings.TrimSuffix(patchOp.Path, "/"+lens.Rename.Source) + "/" + lens.Rename.Destination
		}
	} else if lens.Convert != nil {
		if strings.HasSuffix(patchOp.Path, "/"+lens.Convert.Name) {
			if value, ok := patchOp.Value.(bool); ok {
				for _, mapping := range lens.Convert.Mapping {
					for k, v := range mapping {
						if string(strconv.FormatBool(value)) == k {
							patchOp.Value = v
						}
					}
				}
			} else if value, ok := patchOp.Value.(string); ok {
				for _, mapping := range lens.Convert.Mapping {
					for k, v := range mapping {
						if value == k {
							patchOp.Value = v
						}
					}
				}
			}
		}
	} else if lens.Head != nil {
		if strings.HasSuffix(patchOp.Path, "/"+lens.Head.Name) {
			if slice, ok := patchOp.Value.([]interface{}); ok && len(slice) > 0 {
				patchOp.Value = slice[0]
			}
		}
	} else if lens.In != nil {
		if strings.HasPrefix(patchOp.Path, "/"+lens.In.Name) {
			if arr, ok := patchOp.Value.([]interface{}); ok {
				for i := range arr {
					if obj, ok := arr[i].(map[string]interface{}); ok {
						jsonBytes, err := json.Marshal(obj)
						if err != nil {
							panic(err)
						}
						nestedpatchOpes := DocToPatch(jsonBytes)
						for _, nestedpatchOp := range nestedpatchOpes {
							nestedpatchOp.Path = patchOp.Path + "/" + strconv.Itoa(i) + nestedpatchOp.Path
							for _, nestedLens := range lens.In.Lens {
								nestedpatchOp = applyLens(nestedpatchOp, nestedLens)
							}
						}
						arr[i] = PatchToDoc(nestedpatchOpes)
					}
				}
				patchOp.Value = arr
			} else if obj, ok := patchOp.Value.(map[string]interface{}); ok {
				jsonBytes, err := json.Marshal(obj)
				if err != nil {
					panic(err)
				}
				nestedpatchOpes := DocToPatch(jsonBytes)
				var appliedNestedpatchOpes Patch
				for _, nestedpatchOp := range nestedpatchOpes {
					for _, nestedLens := range lens.In.Lens {
						nestedpatchOp = applyLens(nestedpatchOp, nestedLens)
						appliedNestedpatchOpes = append(appliedNestedpatchOpes, nestedpatchOp)
					}
				}
				patchOp.Value = PatchToDoc(appliedNestedpatchOpes).Data()
			}
		}
	} else if lens.Hoist != nil {
		if strings.HasPrefix(patchOp.Path, "/"+lens.Hoist.Host) {
			if nestedData, ok := patchOp.Value.(map[string]interface{}); ok {
				if hoistValue, ok := nestedData[lens.Hoist.Name]; ok {
					patchOp.Path = strings.TrimPrefix(patchOp.Path, "/"+lens.Hoist.Host) + "/" + lens.Hoist.Name
					patchOp.Value = hoistValue
				}
			}
		}
	} else if lens.Remove != nil {
		if strings.HasSuffix(patchOp.Path, "/"+lens.Remove.Name) {
			return PatchOperation{} // Return an empty patchOp to remove the field
		}
	}

  var newLensSource LensSource
  existingLensSource := patchOp.LensSource

  if existingLensSource != nil {
    newLensSource = append(*existingLensSource, lens)
  } else {
    newLensSource = LensSource{lens}
  }

  patchOp.LensSource = &newLensSource
	return patchOp
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
