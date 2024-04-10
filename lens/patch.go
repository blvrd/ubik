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
	// fmt.Printf("Original document: %s\n", original)
	// fmt.Printf("Modified document: %s\n", modified)

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
	evolvedPatch := ApplyLensToPatch(ls, patch, nil)
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

func ApplyLensToPatch(ls LensSource, p Patch, debug *string) Patch {
	var newPatch Patch

	for _, patchOp := range p {
		JSONcopy, err := gabs.ParseJSON(patchOp.JSON.Bytes())
		newPatchOp := PatchOperation{JSON: JSONcopy}
		if err != nil {
			panic(err)
		}

		for _, lens := range ls {
			switch lens.Op() {
			case "rename":
        if newPatchOp.Path() == "" {
          break
        }

        if debug != nil {
          fmt.Println("DEBUGGING!!!!!!!!!!!!!!!")
          fmt.Printf("newPatchOp path: %s\n\n", newPatchOp.Path())
          fmt.Printf("lens source name: %s\n\n", lens.Source())
        }
				if (newPatchOp.Op() == "replace" || newPatchOp.Op() == "add") && strings.Split(newPatchOp.Path(), "/")[1] == lens.Source() {
					destination := lens.Destination()
          fmt.Println("UGHHHHHHHHHHHHHHHHHHHHHHHHHHHH")
          fmt.Println(destination)
          fmt.Println(lens.Source())
					str := fmt.Sprintf("/%s", destination)
					newPatchOp.SetPath(&str)
					newPatch = append(newPatch, newPatchOp)
				}
			case "remove":
        if newPatchOp.Path() == "" {
          break
        }
				if strings.Split(newPatchOp.Path(), "/")[1] == lens.Name() {
					// fmt.Printf("Remove: %s\n", patchOp)
					newPatchOp.Clear()
					newPatch = append(newPatch, newPatchOp)
				}
			case "head":
        if newPatchOp.Path() == "" {
          break
        }
				if strings.Split(newPatchOp.Path(), "/")[1] == lens.Name() {
					head := newPatchOp.Value().([]any)[0]
					newPatchOp.SetValue(&head)
					newPatch = append(newPatch, newPatchOp)
				}
			case "in":
        if newPatchOp.Path() == "" {
          break
        }

				if strings.Split(newPatchOp.Path(), "/")[1] == lens.Name() {
          fmt.Printf("AAAAAAAAAAAAAAAAAAAAAAAAAA: %s\n", newPatchOp.Value().(map[string]any))
          fmt.Printf("BBBBBBBBBBBBBBBBBBBBBBBBBB: %s\n", lens.Name())
          fmt.Printf("CCCCCCCCCCCCCCCCCCCCCCCCCC: %s\n", lens.NestedLensSource()[0].Source())
          fmt.Printf("CCCCCCCCCCCCCCCCCCCCCCCCCC: %s\n", lens.NestedLensSource()[0].Destination())
					str := strings.Replace(
						newPatchOp.Path(),
						fmt.Sprintf("/%s", lens.Name()),
						fmt.Sprintf("/%s", lens.NestedLensSource()[0].Source()),
						1,
					)

					newPatchOp.SetPath(&str)

          // fmt.Printf("newPatchOp path: %#v\n", newPatchOp.Path())
          // fmt.Printf("newPatchOp path should be: %#v\n", str)
          debugstr := "recursing"
					childPatch := ApplyLensToPatch(
						lens.NestedLensSource(),
						Patch{newPatchOp},
            &debugstr,
					)

					// newPatch = append(newPatch, newPatchOp)
          fmt.Printf("childPatch: %#v\n\n", childPatch[0].JSON)
				}
			case "convert":
				if newPatchOp.Op() != "add" && newPatchOp.Op() != "replace" {
					break
				}

				if fmt.Sprintf("/%s", lens.Name()) != newPatchOp.Path() {
					break
				}

				stringifiedValue := string(newPatchOp.Value().(string))
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
				// fmt.Printf("IMPLEMENT: %s\n", lens.Op())
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
