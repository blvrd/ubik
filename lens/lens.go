package lens

import (
	"encoding/json"
	// "fmt"
)

type Lens struct {
	Rename  *Rename  `json:"rename,omitempty"`
	Convert *Convert `json:"convert,omitempty"`
	Head    *Head    `json:"head,omitempty"`
	In      *In      `json:"in,omitempty"`
	Hoist   *Hoist   `json:"hoist,omitempty"`
	Remove  *Remove  `json:"remove,omitempty"`
	Add     *Add     `json:"add,omitempty"`
}

type Rename struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type Convert struct {
	Name    string           `json:"name"`
	Mapping []map[string]any `json:"mapping"`
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
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default string `json:"default,omitempty"`
}

type Add struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default string `json:"default,omitempty"`
	// Items   []any `json:"items,omitempty"`
}

func (lens Lens) Reverse() Lens {
	if lens.Remove != nil {
		operation := Add{
			Name:    lens.Remove.Name,
			Type:    lens.Remove.Type,
			Default: lens.Remove.Default,
		}

		return Lens{Add: &operation}
	}
	return lens
}

type LensSource []Lens

func (ls LensSource) Reverse() LensSource {
	var reverse LensSource

	for _, lens := range ls {
		reverse = append(reverse, lens.Reverse())
	}

	return reverse
}

func NewLensSource(jsonData []byte) LensSource {
	var ls LensSource

	err := json.Unmarshal(jsonData, &ls)
	if err != nil {
		panic(err)
	}

	return ls
}
