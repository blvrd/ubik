package lens

import (
	"encoding/json"
  "reflect"
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


func (l Lens) Equals(other Lens) bool {
    if (l.Rename == nil && other.Rename != nil) || (l.Rename != nil && other.Rename == nil) {
        return false
    }
    if l.Rename != nil && other.Rename != nil && !l.Rename.Equals(*other.Rename) {
        return false
    }

    if (l.Convert == nil && other.Convert != nil) || (l.Convert != nil && other.Convert == nil) {
        return false
    }
    if l.Convert != nil && other.Convert != nil && !l.Convert.Equals(*other.Convert) {
        return false
    }

    if (l.Head == nil && other.Head != nil) || (l.Head != nil && other.Head == nil) {
        return false
    }
    if l.Head != nil && other.Head != nil && !l.Head.Equals(*other.Head) {
        return false
    }

    if (l.In == nil && other.In != nil) || (l.In != nil && other.In == nil) {
        return false
    }
    if l.In != nil && other.In != nil && !l.In.Equals(*other.In) {
        return false
    }

    if (l.Hoist == nil && other.Hoist != nil) || (l.Hoist != nil && other.Hoist == nil) {
        return false
    }
    if l.Hoist != nil && other.Hoist != nil && !l.Hoist.Equals(*other.Hoist) {
        return false
    }

    if (l.Remove == nil && other.Remove != nil) || (l.Remove != nil && other.Remove == nil) {
        return false
    }
    if l.Remove != nil && other.Remove != nil && !l.Remove.Equals(*other.Remove) {
        return false
    }

    if (l.Add == nil && other.Add != nil) || (l.Add != nil && other.Add == nil) {
        return false
    }
    if l.Add != nil && other.Add != nil && !l.Add.Equals(*other.Add) {
        return false
    }

    return true
}

func (r Rename) Equals(other Rename) bool {
    return r.Source == other.Source && r.Destination == other.Destination
}

func (c Convert) Equals(other Convert) bool {
    if c.Name != other.Name {
        return false
    }
    if len(c.Mapping) != len(other.Mapping) {
        return false
    }
    for i := range c.Mapping {
        if !reflect.DeepEqual(c.Mapping[i], other.Mapping[i]) {
            return false
        }
    }
    return true
}

func (h Head) Equals(other Head) bool {
    return h.Name == other.Name
}

func (i In) Equals(other In) bool {
    if i.Name != other.Name {
        return false
    }
    if len(i.Lens) != len(other.Lens) {
        return false
    }
    for j := range i.Lens {
        if !i.Lens[j].Equals(other.Lens[j]) {
            return false
        }
    }
    return true
}

func (h Hoist) Equals(other Hoist) bool {
    return h.Name == other.Name && h.Host == other.Host
}

func (r Remove) Equals(other Remove) bool {
    return r.Name == other.Name && r.Type == other.Type && r.Default == other.Default
}

func (a Add) Equals(other Add) bool {
    return a.Name == other.Name && a.Type == other.Type && a.Default == other.Default
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
