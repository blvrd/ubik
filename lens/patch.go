package lens

import (
	// "encoding/json"
	// "fmt"

	"fmt"

	"github.com/Jeffail/gabs/v2"
	// "github.com/charmbracelet/log"
)

func ApplyLensToPatch(ls LensSource, p Patch) Patch {
	for _, lens := range ls {
		for _, patchOp := range p {
			switch lens.Op() {
			case "rename":
				destination := lens.Destination()
        str := fmt.Sprintf("/%s", destination)
				patchOp.SetPath(&str)
			case "remove":
				patchOp.Clear()
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
