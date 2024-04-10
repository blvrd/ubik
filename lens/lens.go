package lens

import (
	"encoding/json"
	// "fmt"

	// "fmt"
	// "github.com/Jeffail/gabs/v2"
	// "github.com/charmbracelet/log"
)

type Lens struct {
	Rename  *Rename  `json:"rename,omitempty"`
	Convert *Convert `json:"convert,omitempty"`
	Head    *Head    `json:"head,omitempty"`
	In      *In      `json:"in,omitempty"`
	Hoist   *Hoist   `json:"hoist,omitempty"`
	Remove  *Remove  `json:"remove,omitempty"`
}

type Rename struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type Convert struct {
	Name    string              `json:"name"`
	Mapping []map[string]string `json:"mapping"`
}

type Head struct {
	Name string `json:"name"`
}

type In struct {
	Name string `json:"name"`
	Lens []Lens `json:"lens"`
}

type Hoist struct {
	Name string `json:"name"`
	Host string `json:"host"`
}

type Remove struct {
	Name string `json:"name"`
}

type LensSource []Lens

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

//	type Lens interface {
//		Op() string
//	}
// type Lens struct {
// 	JSON       *gabs.Container
// 	lensSource *LensSource
// }
//
// func (lens Lens) Op() string {
// 	var op string
// 	for key := range lens.JSON.ChildrenMap() {
// 		op = key
// 	}
// 	return op
// }
//
// func (lens *Lens) Destination() string {
// 	return lens.JSON.Search(lens.Op(), "destination").Data().(string)
// }
//
// func (lens *Lens) Source() string {
// 	return lens.JSON.Search(lens.Op(), "source").Data().(string)
// }
//
// func (lens *Lens) Name() string {
// 	return lens.JSON.Search(lens.Op(), "name").Data().(string)
// }
//
// func (lens *Lens) Mapping() any {
// 	return lens.JSON.Search(lens.Op(), "mapping").Data().(any)
// }
//
// func (lens *Lens) NestedLensSource() LensSource {
//   search := lens.JSON.Search(lens.Op(), "lens")
// 	return NewLensSource(search)
// }

func NewLensSource(jsonData []byte) LensSource {
  var ls LensSource

  err := json.Unmarshal(jsonData, &ls)
  if err != nil {
    panic(err)
  }

  return ls
}
