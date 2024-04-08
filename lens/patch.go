package lens

import (
	"encoding/json"
	"strings"
	// "fmt"

	"fmt"

	"github.com/Jeffail/gabs/v2"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/wI2L/jsondiff"
	// "github.com/charmbracelet/log"
)

func PatchToDoc(p Patch) *gabs.Container {
	var original []byte
	var filtered Patch

	for _, patchOp := range p {
		if patchOp.Op() != "noop" {
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
	// modified := []byte(`{"category":"bug","description":"I'm having a problem with this.","id":1,"status":"todo","title":"Found a bug"}`)
	fmt.Printf("Original document: %s\n", original)
	fmt.Printf("Modified document: %s\n", modified)

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
	evolvedPatch := ApplyLensToPatch(ls, patch)
	fmt.Printf("patch: %s\n\n", patch)
	fmt.Printf("evolved patch: %s\n\n", evolvedPatch)
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
	parsedJSON, err := gabs.ParseJSON(patchJSON)
	if err != nil {
		panic(err)
	}

	return NewPatchFromJSON(parsedJSON)
}

func ApplyLensToPatch(ls LensSource, p Patch) Patch {
	var newPatch Patch

	for _, lens := range ls {
		for _, patchOp := range p {
      newPatchOp := PatchOperation{JSON: gabs.Wrap(patchOp.JSON.Data())}
      fmt.Printf("using new copy: %v\n", patchOp.JSON != newPatchOp.JSON)
			switch lens.Op() {
			case "rename":
				if (patchOp.Op() == "replace" || patchOp.Op() == "add") && strings.Split(patchOp.Path(), "/")[1] == lens.Source() {
					destination := lens.Destination()
					str := fmt.Sprintf("/%s", destination)
					newPatchOp.SetPath(&str)
          newPatch = append(newPatch, newPatchOp)
				}
			case "remove":
				if strings.Split(patchOp.Path(), "/")[1] == lens.Name() {
					newPatchOp.Clear()
          newPatch = append(newPatch, newPatchOp)
				}
			case "head":
				if strings.Split(patchOp.Path(), "/")[1] == lens.Name() {
					head := patchOp.Value().([]any)[0]
					newPatchOp.SetValue(&head)
          newPatch = append(newPatch, newPatchOp)
				}
			case "in":
				if strings.Split(patchOp.Path(), "/")[1] == lens.Name() {
					fmt.Printf("nested lens: %#v\n", lens.NestedLensSource())
          newPatch = append(newPatch, newPatchOp)
				}
				// case "hoist":
			case "convert":
				if patchOp.Op() != "add" && patchOp.Op() != "replace" {
					break
				}

				if fmt.Sprintf("/%s", lens.Name()) != patchOp.Path() {
					break
				}

				stringifiedValue := string(patchOp.Value().(string))
				var newValue any
				for _, mapping := range lens.Mapping().([]any) {
					m := mapping.(map[string]any)
					if _, exists := m[stringifiedValue]; !exists {
						break
					} else {
						newValue = m[stringifiedValue]
					}
				}

			  newPatchOp.SetValue(&newValue)
        newPatch = append(newPatch, newPatchOp)
			default:
        newPatch = append(newPatch, newPatchOp)
				fmt.Printf("IMPLEMENT: %s\n", lens.Op())
			}
		}
	}
	return newPatch
}

type Patch []PatchOperation

// type PatchOperation interface {
// 	Op() string
// 	Path() string
//   SetPath(string) error
// 	json.Marshaler
// 	json.Unmarshaler
// }

type PatchOperation struct {
	JSON *gabs.Container
}

func (p *PatchOperation) Path() string {
	return p.JSON.Search("path").Data().(string)
}

func (p *PatchOperation) SetPath(path *string) error {
	if path != nil {
		_, err := p.JSON.SetP(*path, "path")
		if err != nil {
			return err
		}
	} else {
		err := p.JSON.Delete("path")
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PatchOperation) Clear() {
	p.SetOp("noop")
	p.SetPath(nil)
	p.SetValue(nil)
}

func (p PatchOperation) Op() string {
	return p.JSON.Search("op").Data().(string)
}

func (p *PatchOperation) SetOp(op string) error {
	_, err := p.JSON.SetP(op, "op")
	if err != nil {
		return err
	}
	return nil
}

func (p *PatchOperation) Value() any {
	return p.JSON.Search("value").Data().(any)
}

func (p *PatchOperation) SetSource(source *string) error {
	if source != nil {
		_, err := p.JSON.SetP(source, "source")
		if err != nil {
			return err
		}
	} else {
		err := p.JSON.Delete("source")
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PatchOperation) SetValue(value *any) error {
	if value != nil {
		_, err := p.JSON.SetP(*value, "value")

		if err != nil {
			return err
		}
	} else {
		err := p.JSON.Delete("value")
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PatchOperation) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(p.JSON)

	return b, err
}

func NewPatchOperationFromJSON(c *gabs.Container) PatchOperation {
	return PatchOperation{JSON: c}
}

func NewPatchFromJSON(c *gabs.Container) Patch {
	var patch Patch

	for _, item := range c.Children() {
		patchOp := NewPatchOperationFromJSON(item)
		patch = append(patch, patchOp)
	}

	return patch
}
