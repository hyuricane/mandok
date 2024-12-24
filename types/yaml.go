package types

import (
	"encoding/json"
	"log"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProjectConfig struct {
	Version  string                   `yaml:"version,omitempty" json:"version,omitempty"`
	Services map[string]ServiceConfig `yaml:"services,omitempty" json:"services,omitempty"`
}

type ServiceConfig struct {
	Image       string       `yaml:"image,omitempty" json:"image,omitempty"`
	Environment *MapButArray `yaml:"environment,omitempty" json:"environment,omitempty"`
	Labels      *MapButArray `yaml:"labels,omitempty" json:"labels,omitempty"`
	Ports       []string     `yaml:"ports,omitempty" json:"ports,omitempty"`
	Volumes     []string     `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Links       []string     `yaml:"links,omitempty" json:"links,omitempty"`
	DependsOn   []string     `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Command     string       `yaml:"command,omitempty" json:"command,omitempty"`
}

type MapButArray struct {
	m       map[string]string
	yamlTag string
}

func (m *MapButArray) UnmarshalYAML(val *yaml.Node) error {
	if m == nil {
		m = &MapButArray{
			m: map[string]string{},
		}
	}
	if m.m == nil {
		m.m = map[string]string{}
	}
	m.yamlTag = val.Tag
	if val.Tag == "!!map" {
		for i := 0; i < len(val.Content); i += 2 {
			k := val.Content[i].Value
			v := val.Content[i+1].Value
			m.m[k] = v
		}
	} else if val.Tag == "!!seq" {
		for i := 0; i < len(val.Content); i++ {
			vals := strings.Split(val.Content[i].Value, "=")
			k := vals[0]
			v := ""
			if len(vals) > 1 {
				v = vals[1]
			}
			m.m[k] = v
		}
	}
	return nil
}

func (m MapButArray) MarshalYAML() (interface{}, error) {
	if len(m.m) == 0 {
		return nil, nil
	}
	switch m.yamlTag {
	case "!!map":
		return m.m, nil
	case "!!seq":
		mseq := []string{}
		for k, v := range m.m {
			if v == "" {
				mseq = append(mseq, k)
			} else {
				mseq = append(mseq, k+"="+v)
			}
		}
		return mseq, nil
	}
	return nil, nil
}

func (m *MapButArray) UnmarshalJSON(b []byte) error {
	if m == nil {
		m = &MapButArray{
			m: map[string]string{},
		}
	}
	if m.m == nil {
		m.m = map[string]string{}
	}
	var i interface{} = nil
	if err := json.Unmarshal(b, &i); err != nil {
		log.Printf("[ERROR] %v", err)
		return err
	}
	if i == nil {
		return nil
	}
	switch val := i.(type) {
	case map[string]interface{}:
		for k, v := range val {
			m.m[k] = v.(string)
		}
	case []interface{}:
		for _, v := range val {
			vals := strings.Split(v.(string), "=")
			k := vals[0]
			v := ""
			if len(vals) > 1 {
				v = vals[1]
			}
			m.m[k] = v
		}
	}
	return nil
}

func (m MapButArray) MarshalJSON() ([]byte, error) {
	if len(m.m) == 0 {
		return nil, nil
	}
	return json.Marshal(m.m)
}

func (m *MapButArray) Set(key, val string) {
	if m.m == nil {
		m.m = map[string]string{}
	}
	m.m[key] = val
}

func (m *MapButArray) SetYamlTag(tag string) {
	m.yamlTag = tag
}
