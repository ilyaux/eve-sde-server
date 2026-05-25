package sde

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser reads the YAML files from an extracted EVE SDE archive.
type Parser struct {
	root string
}

// NewParser creates a parser rooted at an extracted SDE directory.
func NewParser(root string) *Parser {
	return &Parser{root: root}
}

// Type represents an EVE inventory type imported from SDE.
type Type struct {
	ID          int
	Name        string
	Description string
	Volume      float64
	GroupID     int
	CategoryID  int
	Published   bool
}

// Group represents an EVE inventory group imported from SDE.
type Group struct {
	ID         int
	CategoryID int
	Name       string
	Published  bool
}

// Category represents an EVE inventory category imported from SDE.
type Category struct {
	ID        int
	Name      string
	Published bool
}

type typeYAML struct {
	Name        localizedString `yaml:"name"`
	Description localizedString `yaml:"description"`
	Volume      float64         `yaml:"volume"`
	GroupID     int             `yaml:"groupID"`
	Published   *bool           `yaml:"published"`
}

type groupYAML struct {
	Name       localizedString `yaml:"name"`
	CategoryID int             `yaml:"categoryID"`
	Published  *bool           `yaml:"published"`
}

type categoryYAML struct {
	Name      localizedString `yaml:"name"`
	Published *bool           `yaml:"published"`
}

type localizedString string

func (s *localizedString) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		*s = localizedString(value.Value)
	case yaml.MappingNode:
		var fallback string
		for i := 0; i+1 < len(value.Content); i += 2 {
			key := value.Content[i]
			val := value.Content[i+1]
			if val.Kind != yaml.ScalarNode {
				continue
			}
			if key.Value == "en" {
				*s = localizedString(val.Value)
				return nil
			}
			if fallback == "" {
				fallback = val.Value
			}
		}
		*s = localizedString(fallback)
	default:
		*s = ""
	}

	return nil
}

func (s localizedString) String() string {
	return strings.TrimSpace(string(s))
}

// ParseTypes parses fsd/types.yaml from the SDE archive.
func (p *Parser) ParseTypes() ([]Type, error) {
	var raw map[int]typeYAML
	if err := p.readYAML(&raw, "fsd/types.yaml", "fsd/typeIDs.yaml", "types.yaml", "typeIDs.yaml"); err != nil {
		return nil, fmt.Errorf("parse types: %w", err)
	}

	types := make([]Type, 0, len(raw))
	for id, item := range raw {
		name := item.Name.String()
		if name == "" {
			continue
		}
		types = append(types, Type{
			ID:          id,
			Name:        name,
			Description: item.Description.String(),
			Volume:      item.Volume,
			GroupID:     item.GroupID,
			Published:   boolValue(item.Published, true),
		})
	}
	sort.Slice(types, func(i, j int) bool { return types[i].ID < types[j].ID })

	return types, nil
}

// ParseGroups parses fsd/groups.yaml from the SDE archive.
func (p *Parser) ParseGroups() ([]Group, error) {
	var raw map[int]groupYAML
	if err := p.readYAML(&raw, "fsd/groups.yaml", "fsd/groupIDs.yaml", "groups.yaml", "groupIDs.yaml"); err != nil {
		return nil, fmt.Errorf("parse groups: %w", err)
	}

	groups := make([]Group, 0, len(raw))
	for id, item := range raw {
		name := item.Name.String()
		if name == "" {
			continue
		}
		groups = append(groups, Group{
			ID:         id,
			CategoryID: item.CategoryID,
			Name:       name,
			Published:  boolValue(item.Published, true),
		})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].ID < groups[j].ID })

	return groups, nil
}

// ParseCategories parses fsd/categories.yaml from the SDE archive.
func (p *Parser) ParseCategories() ([]Category, error) {
	var raw map[int]categoryYAML
	if err := p.readYAML(&raw, "fsd/categories.yaml", "fsd/categoryIDs.yaml", "categories.yaml", "categoryIDs.yaml"); err != nil {
		return nil, fmt.Errorf("parse categories: %w", err)
	}

	categories := make([]Category, 0, len(raw))
	for id, item := range raw {
		name := item.Name.String()
		if name == "" {
			continue
		}
		categories = append(categories, Category{
			ID:        id,
			Name:      name,
			Published: boolValue(item.Published, true),
		})
	}
	sort.Slice(categories, func(i, j int) bool { return categories[i].ID < categories[j].ID })

	return categories, nil
}

func (p *Parser) readYAML(out interface{}, candidates ...string) error {
	path, err := p.findFile(candidates...)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}

	return nil
}

func (p *Parser) findFile(candidates ...string) (string, error) {
	for _, candidate := range candidates {
		path := filepath.Join(p.root, filepath.FromSlash(candidate))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}

		path = filepath.Join(p.root, "sde", filepath.FromSlash(candidate))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}

	var found string
	lowerCandidates := make([]string, len(candidates))
	for i, candidate := range candidates {
		lowerCandidates[i] = strings.ToLower(filepath.ToSlash(candidate))
	}

	err := filepath.WalkDir(p.root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return nil
		}

		lowerPath := strings.ToLower(filepath.ToSlash(path))
		for _, candidate := range lowerCandidates {
			if strings.HasSuffix(lowerPath, "/"+candidate) || strings.EqualFold(entry.Name(), filepath.Base(candidate)) {
				found = path
				return fs.SkipAll
			}
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk SDE dir: %w", err)
	}
	if found == "" {
		return "", fmt.Errorf("none of %s found under %s", strings.Join(candidates, ", "), p.root)
	}

	return found, nil
}

func boolValue(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}
