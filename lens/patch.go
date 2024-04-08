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

func PatchToDoc(p Patch, optionalDoc *gabs.Container) *gabs.Container {
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
	if optionalDoc != nil {
    fmt.Printf("patch in patchtodoc: %s\n\n", p)
    original = optionalDoc.Bytes()
	} else {
    original = []byte(`{}`)
	}
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
  fmt.Printf("doc: %s\n\n", string(original))
	patch := DocToPatch(original)
  fmt.Printf("patch: %s\n\n", patch)
  evolvedPatch := ApplyLensToPatch(ls, patch)
  fmt.Printf("evolved patch: %s\n\n", evolvedPatch)
  x := PatchToDoc(evolvedPatch, nil)
  fmt.Printf("heyyyy there: %s", x)
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

	return NewPatch(parsedJSON)
}

func ApplyLensToPatch(ls LensSource, p Patch) Patch {
	for _, lens := range ls {
		for _, patchOp := range p {
      // fmt.Printf("lens op: %s\n\n", lens.Op())
			switch lens.Op() {
			case "rename":
        if (patchOp.Op() == "replace" || patchOp.Op() == "add") && strings.Split(patchOp.Path(), "/")[1] == lens.Source() {
          destination := lens.Destination()
          str := fmt.Sprintf("/%s", destination)
          patchOp.SetPath(&str)
        }
			case "remove":
        if strings.Split(patchOp.Path(), "/")[1] == lens.Name() {
          patchOp.Clear()
        }
      default:
			}
		}
	}
	return p
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

func (p *PatchOperation) SetValue(value *string) error {
	if value != nil {
		_, err := p.JSON.SetP(value, "value")
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

func NewPatchOperation(c *gabs.Container) PatchOperation {
	return PatchOperation{JSON: c}
}

func NewPatch(c *gabs.Container) Patch {
	var patch Patch

	for _, item := range c.Children() {
		patchOp := NewPatchOperation(item)
		patch = append(patch, patchOp)
	}

	return patch
}
