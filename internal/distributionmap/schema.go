package distributionmap

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yukihito-jokyu/context-cli/internal/distribution"
	"go.yaml.in/yaml/v3"
)

const schemaVersion = 1

type schema struct {
	SchemaVersion int                        `yaml:"schema_version"`
	Workspaces    map[string]workspaceSchema `yaml:"workspaces"`
}

type workspaceSchema struct {
	Project      string        `yaml:"project"`
	Destinations []string      `yaml:"destinations"`
	Skills       []skillSchema `yaml:"skills"`
}

type skillSchema struct {
	Name         string `yaml:"name"`
	Source       string `yaml:"source"`
	Destination  string `yaml:"destination"`
	RelativePath string `yaml:"relative_path"`
	Hash         string `yaml:"hash"`
}

func encode(records map[string]distribution.WorkspaceRecord) ([]byte, error) {
	document := schema{SchemaVersion: schemaVersion, Workspaces: make(map[string]workspaceSchema, len(records))}
	for workspace, record := range records {
		if record.WorkspaceRoot != workspace {
			return nil, newError("encode", ErrSchema, nil)
		}
		converted := workspaceSchema{Project: record.Project}
		for _, destination := range record.Destinations {
			converted.Destinations = append(converted.Destinations, string(destination))
		}
		for _, skill := range record.Skills {
			converted.Skills = append(converted.Skills, skillSchema{
				Name: skill.Name, Source: string(skill.Source), Destination: string(skill.Destination),
				RelativePath: filepath.ToSlash(skill.RelativePath), Hash: skill.Hash,
			})
		}
		normalizeWorkspace(&converted)
		document.Workspaces[workspace] = converted
	}
	if err := validateSchema(document); err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(document)
	if err != nil {
		return nil, newError("encode", ErrIO, err)
	}
	return data, nil
}

func decode(data []byte) (map[string]distribution.WorkspaceRecord, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var document schema
	if err := decoder.Decode(&document); err != nil {
		return nil, newError("decode", ErrSchema, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, newError("decode", ErrSchema, err)
	}
	if err := validateSchema(document); err != nil {
		return nil, err
	}
	records := make(map[string]distribution.WorkspaceRecord, len(document.Workspaces))
	for workspace, value := range document.Workspaces {
		normalizeWorkspace(&value)
		record := distribution.WorkspaceRecord{
			WorkspaceRoot: workspace,
			Project:       value.Project,
		}
		for _, destination := range value.Destinations {
			record.Destinations = append(record.Destinations, distribution.Destination(destination))
		}
		for _, skill := range value.Skills {
			record.Skills = append(record.Skills, distribution.SkillRecord{
				Name: skill.Name, Source: distribution.SkillSource(skill.Source),
				Destination:  distribution.Destination(skill.Destination),
				RelativePath: skill.RelativePath, Hash: skill.Hash,
			})
		}
		records[workspace] = record
	}
	return records, nil
}

//nolint:gocognit,cyclop // スキーマ内の集合整合性と複合一意制約を同時に検証します。
func validateSchema(document schema) error {
	if document.SchemaVersion != schemaVersion || document.Workspaces == nil {
		return newError("validate", ErrSchema, nil)
	}
	for workspace, record := range document.Workspaces {
		if !filepath.IsAbs(workspace) || filepath.Clean(workspace) != workspace || !validName(record.Project) {
			return newError("validate", ErrSchema, nil)
		}
		destinationSet := make(map[string]struct{}, len(record.Destinations))
		for _, destination := range record.Destinations {
			if destination != string(distribution.DestinationCodex) &&
				destination != string(distribution.DestinationClaude) {
				return newError("validate", ErrSchema, nil)
			}
			if _, exists := destinationSet[destination]; exists {
				return newError("validate", ErrSchema, nil)
			}
			destinationSet[destination] = struct{}{}
		}
		skillDestinations := make(map[string]struct{})
		seenSkills := make(map[string]struct{}, len(record.Skills))
		for _, skill := range record.Skills {
			if !validName(skill.Name) ||
				(skill.Source != string(distribution.SkillSourceProject) &&
					skill.Source != string(distribution.SkillSourceCommon)) ||
				(skill.Destination != string(distribution.DestinationCodex) &&
					skill.Destination != string(distribution.DestinationClaude)) ||
				!validHash(skill.Hash) ||
				!validRelativePath(skill) {
				return newError("validate", ErrSchema, nil)
			}
			key := skill.Name + "\x00" + skill.Destination
			if _, exists := seenSkills[key]; exists {
				return newError("validate", ErrSchema, nil)
			}
			seenSkills[key] = struct{}{}
			skillDestinations[skill.Destination] = struct{}{}
		}
		if len(destinationSet) != len(skillDestinations) {
			return newError("validate", ErrSchema, nil)
		}
		for destination := range destinationSet {
			if _, exists := skillDestinations[destination]; !exists {
				return newError("validate", ErrSchema, nil)
			}
		}
	}
	return nil
}

func validRelativePath(skill skillSchema) bool {
	expectedPrefix := ""
	switch distribution.Destination(skill.Destination) {
	case distribution.DestinationCodex:
		expectedPrefix = ".codex"
	case distribution.DestinationClaude:
		expectedPrefix = ".claude"
	default:
		return false
	}
	expected := filepath.ToSlash(filepath.Join(expectedPrefix, "skills", skill.Name))
	return skill.RelativePath == expected && !filepath.IsAbs(skill.RelativePath)
}

func validHash(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validName(value string) bool {
	return value != "" && value != "." && value != ".." &&
		!strings.Contains(value, "/") && !strings.Contains(value, "\\")
}

func normalizeWorkspace(record *workspaceSchema) {
	sort.Strings(record.Destinations)
	sort.Slice(record.Skills, func(i, j int) bool {
		if record.Skills[i].Name != record.Skills[j].Name {
			return record.Skills[i].Name < record.Skills[j].Name
		}
		if record.Skills[i].Destination != record.Skills[j].Destination {
			return record.Skills[i].Destination < record.Skills[j].Destination
		}
		return record.Skills[i].RelativePath < record.Skills[j].RelativePath
	})
}
