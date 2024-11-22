package sbswagger

import (
	"bytes"
	"encoding/json"
	"sort"
)

func Merge(specs ...[]byte) ([]byte, error) {
	roots := make([]Root, 0)
	for _, spec := range specs {
		var root Root
		err := json.Unmarshal(spec, &root)
		if err != nil {
			return nil, err
		}
		roots = append(roots, root)
	}

	fullRoot := &roots[0]
	for i := 1; i < len(roots); i++ {
		fullRoot = fullRoot.Merge(&roots[i])
	}

	fullRoot.processTags()
	fullRoot.sort()

	fullRoot.Host = "localhost:3000"
	fullRoot.Schemes = []string{"http"}
	fullRoot.Info["title"] = config.Title
	fullRoot.Info["version"] = config.Version

	data, err := json.Marshal(fullRoot)
	if err != nil {
		return nil, err
	}

	return data, nil
}

type Root struct {
	Swagger     string                 `json:"swagger"`
	Info        map[string]interface{} `json:"info"`
	Consumes    []string               `json:"consumes"`
	Produces    []string               `json:"produces"`
	Paths       Paths                  `json:"paths"`
	Definitions map[string]interface{} `json:"definitions"`
	BasePath    string                 `json:"basePath"`
	Schemes     []string               `json:"schemes"`
	Host        string                 `json:"host"`
	Tags        []Tag                  `json:"tags"`
}

type Tag struct {
	Description string `json:"description"`
	Name        string `json:"name"`
}

func (p *Root) sort() {
	p.Paths.sort()
}

func (p *Root) Merge(other *Root) *Root {
	p.Paths = p.Paths.Merge(other.Paths)
	for k, v := range other.Definitions {
		p.Definitions[k] = v
	}

	return p
}

func (p *Root) processTags() {
	localTags := make(map[string]bool)
	for _, path := range p.Paths.Map {
		for _, op := range path.Map {
			tags, ok := op["tags"]
			if ok {
				typedTags, ok := tags.([]interface{})
				if ok {
					for _, t := range typedTags {
						typedTag, ok := t.(string)
						if ok {
							localTags[typedTag] = true
						}
					}
				}
			}
		}
	}
	tagsList := make([]string, 0)
	for k := range localTags {
		tagsList = append(tagsList, k)
	}
	sort.Strings(tagsList)
	tags := make([]Tag, 0)
	for _, t := range tagsList {
		tags = append(tags, Tag{
			Name: t,
		})
	}
	p.Tags = tags
}

type Paths struct {
	Order []string
	Map   map[string]OperationSet
}

func (p *Paths) sort() {
	sort.Strings(p.Order)
	for _, op := range p.Map {
		op.sort()
	}
}

func (p *Paths) Merge(other Paths) Paths {
	for k, v := range other.Map {
		detail, ok := p.Map[k]
		if !ok {
			p.Map[k] = v
			p.Order = append(p.Order, k)
		} else {
			detail = detail.Merge(v)
			p.Map[k] = detail
		}
	}

	return *p
}

type OperationSet struct {
	Order []string
	Map   map[string]Operation
}

func (p *OperationSet) sort() {
	sort.Strings(p.Order)
}

func (p *OperationSet) Merge(other OperationSet) OperationSet {
	for k, v := range other.Map {
		detail, ok := p.Map[k]
		if !ok {
			p.Map[k] = v
			p.Order = append(p.Order, k)
		} else {
			detail = detail.Merge(v)
			p.Map[k] = detail
		}
	}

	return *p
}

type Operation map[string]interface{}

func (p *Operation) Merge(other Operation) Operation {
	local := *p
	for k, v := range other {
		local[k] = v
	}

	return local
}

// Ordered maps from https://stackoverflow.com/questions/48293036/prevent-alphabetically-ordering-json-at-marshal

func (om *Paths) UnmarshalJSON(b []byte) error {
	json.Unmarshal(b, &om.Map)

	index := make(map[string]int)
	for key := range om.Map {
		om.Order = append(om.Order, key)
		esc, _ := json.Marshal(key) //Escape the key
		index[key] = bytes.Index(b, esc)
	}

	sort.Slice(om.Order, func(i, j int) bool { return index[om.Order[i]] < index[om.Order[j]] })
	return nil
}

func (om Paths) MarshalJSON() ([]byte, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	buf.WriteRune('{')
	l := len(om.Order)
	for i, key := range om.Order {
		km, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(km)
		buf.WriteRune(':')
		vm, err := json.Marshal(om.Map[key])
		if err != nil {
			return nil, err
		}
		buf.Write(vm)
		if i != l-1 {
			buf.WriteRune(',')
		}
	}
	buf.WriteRune('}')
	return buf.Bytes(), nil
}

func (om *OperationSet) UnmarshalJSON(b []byte) error {
	json.Unmarshal(b, &om.Map)

	index := make(map[string]int)
	for key := range om.Map {
		om.Order = append(om.Order, key)
		esc, _ := json.Marshal(key) //Escape the key
		index[key] = bytes.Index(b, esc)
	}

	sort.Slice(om.Order, func(i, j int) bool { return index[om.Order[i]] < index[om.Order[j]] })
	return nil
}

func (om OperationSet) MarshalJSON() ([]byte, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	buf.WriteRune('{')
	l := len(om.Order)
	for i, key := range om.Order {
		km, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(km)
		buf.WriteRune(':')
		vm, err := json.Marshal(om.Map[key])
		if err != nil {
			return nil, err
		}
		buf.Write(vm)
		if i != l-1 {
			buf.WriteRune(',')
		}
	}
	buf.WriteRune('}')
	return buf.Bytes(), nil
}
