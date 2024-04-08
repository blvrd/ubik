package lens

import (
	// "encoding/json"
	// "fmt"

	// "fmt"
	"github.com/Jeffail/gabs/v2"
	// "github.com/charmbracelet/log"
)

// export interface Property {
//   name?: string
//   type: JSONSchema7TypeName | JSONSchema7TypeName[]
//   default?: any
//   required?: boolean
//   items?: Property
// }
//
// export interface AddProperty extends Property {
//   op: 'add'
// }
//
// export interface RemoveProperty extends Property {
//   op: 'remove'
// }

// type Lens interface {
// 	Op() string
// }
//
type Lens struct {
  JSON *gabs.Container
}

func (lens Lens) Op() string {
  var op string
  for key := range lens.JSON.ChildrenMap() {
    op = key
  }
  return op
}

func (lens *Lens) Destination() string {
  return lens.JSON.Search(lens.Op(), "destination").Data().(string)
}

func (lens *Lens) Source() string {
  return lens.JSON.Search(lens.Op(), "source").Data().(string)
}

func (lens *Lens) Name() string {
  return lens.JSON.Search(lens.Op(), "name").Data().(string)
}

func (lens *Lens) Mapping() any {
  return lens.JSON.Search(lens.Op(), "mapping").Data().(any)
}

func NewLens(c *gabs.Container) Lens {
  return Lens{JSON: c}
}

type LensSource []Lens

func NewLensSource(c *gabs.Container) LensSource {
  var lensSource LensSource

  for _, item := range c.Children() {
    lens := NewLens(item)
    lensSource = append(lensSource, lens)
  }

  return lensSource
}
