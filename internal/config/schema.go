package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

const currentSchemaVersion = 1

var errMultipleYAMLDocuments = errors.New("multiple YAML documents")

type schema struct {
	SchemaVersion     int    `yaml:"schema_version"`
	ContextRepository string `yaml:"context_repository"`
}

type legacySchema struct {
	Version        int    `yaml:"version"`
	RepositoryPath string `yaml:"repository_path"`
}

func decodeSchema(data []byte) (schema, error) {
	var value schema
	if err := decodeStrictYAML(data, &value); err == nil {
		if err := validateSchema(value); err != nil {
			return schema{}, err
		}
		return value, nil
	}

	var legacy legacySchema
	if err := decodeStrictYAML(data, &legacy); err != nil {
		return schema{}, newError("decode", ErrFormat, err)
	}
	value = schema{
		SchemaVersion:     legacy.Version,
		ContextRepository: legacy.RepositoryPath,
	}
	if err := validateSchema(value); err != nil {
		return schema{}, err
	}
	return value, nil
}

func decodeStrictYAML(data []byte, value any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("YAMLデコード: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errMultipleYAMLDocuments
		}
		return fmt.Errorf("YAML終端検証: %w", err)
	}
	return nil
}

func encodeSchema(contextRepository string) ([]byte, error) {
	value := schema{
		SchemaVersion:     currentSchemaVersion,
		ContextRepository: contextRepository,
	}
	if err := validateSchema(value); err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(value)
	if err != nil {
		return nil, newError("encode", ErrFormat, err)
	}
	return data, nil
}

func validateSchema(value schema) error {
	if value.SchemaVersion != currentSchemaVersion {
		return newError("validate", ErrSchema, nil)
	}
	if value.ContextRepository == "" || !filepath.IsAbs(value.ContextRepository) {
		return newError("validate", ErrSchema, nil)
	}
	return nil
}
