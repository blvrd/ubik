package lens

import (
	"encoding/json"
	"strconv"
	"strings"

	// "github.com/charmbracelet/log"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/wI2L/jsondiff"
)

func PatchToDoc(p Patch) []byte {
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

	original = []byte(`{}`)
	patch, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		panic(err)
	}

	modified, err := patch.Apply(original)
	if err != nil {
		panic(err)
	}

	return modified
}

func ApplyLensToDoc(ls LensSource, doc []byte) []byte {
  // log.Debug(string(doc))
	original := doc
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
	return NewPatchFromJSON(patchJSON)
}

type PatchOperation struct {
	Op         string      `json:"op"`
	Path       string      `json:"path"`
	Value      interface{} `json:"value,omitempty"`
	LensSource *LensSource `json:"lens,omitempty"`
}

type Patch []PatchOperation

func InterpretLens(patches Patch, lenses LensSource) Patch {
	var result Patch

	for _, lens := range lenses {
		var transformedPatches []PatchOperation

		for _, patch := range patches {
			transformedPatch := applyLens(patch, lens, false)

			if transformedPatch.Op != "" {
				transformedPatches = append(transformedPatches, transformedPatch)
			}
		}

		patches = transformedPatches
	}

	result = patches

	if result == nil {
		return Patch{}
	}
	return result
}

func applyLens(patchOp PatchOperation, lens Lens, recursing bool) PatchOperation {
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
								nestedpatchOp = applyLens(nestedpatchOp, nestedLens, true)
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
				var appliedNestedpatchOps Patch
				for _, nestedpatchOp := range nestedpatchOpes {
					for _, nestedLens := range lens.In.Lens {
						nestedpatchOp = applyLens(nestedpatchOp, nestedLens, true)
						appliedNestedpatchOps = append(appliedNestedpatchOps, nestedpatchOp)
					}
				}
				var newValue map[string]any
				newDoc := PatchToDoc(appliedNestedpatchOps)

				err = json.Unmarshal(newDoc, &newValue)
				if err != nil {
					panic(err)
				}

				patchOp.Value = newValue
			} else {
				newPath := strings.Replace(patchOp.Path, "/"+lens.In.Name, "", 1)
				nestedPatchOp := PatchOperation{
					Op:    patchOp.Op,
					Path:  newPath,
					Value: patchOp.Value,
				}

				// var appliedNestedpatchOp PatchOperation
				for _, nestedLens := range lens.In.Lens {
					nestedPatchOp = applyLens(nestedPatchOp, nestedLens, true)
					// appliedNestedpatchOp = nestedPatchOp
				}

				nestedPatchOp.Path = "/" + lens.In.Name + nestedPatchOp.Path
				patchOp = nestedPatchOp

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
			patchOp.Op = "noop"
		}
	} else if lens.Add != nil {
		if patchOp.Op == "noop" {
			patchOp.Op = "add"
		} else {

		}
	}

	var newLensSource LensSource
	existingLensSource := patchOp.LensSource

	var existingLensSourceAlreadyContainsLens bool

	if !recursing {
		if existingLensSource != nil {
			for _, l := range *existingLensSource {
				if l == lens {
					existingLensSourceAlreadyContainsLens = true
				} else {
					existingLensSourceAlreadyContainsLens = false
				}
			}
		}

		if !existingLensSourceAlreadyContainsLens {
			if existingLensSource != nil {
				newLensSource = append(*existingLensSource, lens)
			} else {
				newLensSource = LensSource{lens}
			}
		}

		if newLensSource != nil {
			patchOp.LensSource = &newLensSource
		}
	}
	return patchOp
}

func NewPatchFromJSON(jsonData []byte) Patch {
	var patch Patch

	err := json.Unmarshal(jsonData, &patch)
	if err != nil {
		panic(err)
	}

	return patch
}
