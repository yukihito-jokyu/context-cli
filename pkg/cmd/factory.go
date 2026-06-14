package cmd

import (
	"io"
	"os"

	"github.com/yukihito-jokyu/context-cli/internal/config"
	"github.com/yukihito-jokyu/context-cli/internal/repository"
)

// Config は CLI 設定のインターフェースを表します。
type Config interface {
	GetContextRepository() string
	SetContextRepository(expected, newPath string) error
}

// RepositoryValidator はContext Repositoryの検証境界を表します。
type RepositoryValidator interface {
	Validate(path string) (string, error)
}

// Factory は CLI の依存関係を管理し、注入します。
type Factory struct {
	IOOut io.Writer
	IOErr io.Writer
	IOIn  io.Reader

	RepositoryValidator RepositoryValidator

	// Config は Config インスタンスを返す関数です（遅延ロードされます）。
	Config func() (Config, error)
}

// NewFactory は標準の入出力（os.Stdout/Stderr/Stdin）を使用して新しい Factory を作成します。
func NewFactory() *Factory {
	environment := config.NewOSEnvironment()
	fileSystem := config.NewOSFileSystem()
	return &Factory{
		IOOut:               os.Stdout,
		IOErr:               os.Stderr,
		IOIn:                os.Stdin,
		RepositoryValidator: repository.NewValidator(repository.NewFileSystem()),
		Config: func() (Config, error) {
			return config.Open(environment, fileSystem)
		},
	}
}
