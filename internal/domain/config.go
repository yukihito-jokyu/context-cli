// Package domain は、外部技術に依存しないドメイン規則を定義します。
package domain

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// CurrentConfigVersion は、サポートされている config.yaml のスキーマバージョンです。
const CurrentConfigVersion = 1

var (
	// ErrInvalidConfig は、不正なYAMLまたは無効な設定値を示します。
	ErrInvalidConfig = errors.New("invalid config")

	// ErrUnsupportedConfigVersion は、サポートされていない config.yaml のスキーマを示します。
	ErrUnsupportedConfigVersion = errors.New("unsupported config version")

	// ErrInvalidRepositoryPath は、リポジトリを特定できないパスを示します。
	ErrInvalidRepositoryPath = errors.New("invalid repository path")
)

// Config は、グローバルなコンテキストリポジトリの設定を保持します。
type Config struct {
	Version        int    `yaml:"version"`
	RepositoryPath string `yaml:"repository_path"`
}

// ParseConfig は、1つの config.yaml ドキュメントを厳格にデコードし、検証します。
func ParseConfig(data []byte) (Config, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("%w: unable to decode", ErrInvalidConfig)
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Config{}, fmt.Errorf("%w: multiple YAML documents", ErrInvalidConfig)
	}

	if err := config.Validate(); err != nil {
		return Config{}, err
	}

	return config, nil
}

// Validate は、スキーマバージョンと正規化されたリポジトリパスを検証します。
func (c Config) Validate() error {
	if c.Version != CurrentConfigVersion {
		return ErrUnsupportedConfigVersion
	}
	if c.RepositoryPath == "" ||
		!filepath.IsAbs(c.RepositoryPath) ||
		filepath.Clean(c.RepositoryPath) != c.RepositoryPath {
		return ErrInvalidConfig
	}

	return nil
}

// NormalizeRepositoryPath は、シンボリックリンクを解決せずに、絶対的な辞書順パスを返します。
func NormalizeRepositoryPath(path string) (string, error) {
	if path == "" {
		return "", ErrInvalidRepositoryPath
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("%w: unable to make path absolute", ErrInvalidRepositoryPath)
	}

	return filepath.Clean(absolutePath), nil
}
