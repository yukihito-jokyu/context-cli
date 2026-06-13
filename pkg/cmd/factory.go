package cmd

import (
	"io"
	"os"
)

// Config は CLI 設定のインターフェースを表します。
type Config interface {
	GetContextRepository() string
	SetContextRepository(path string) error
}

// dummyConfig はテストおよびフォールバック用のシンプルなメモリ内 Config 実装です。
type dummyConfig struct {
	repoPath string
}

func (c *dummyConfig) GetContextRepository() string {
	return c.repoPath
}

func (c *dummyConfig) SetContextRepository(path string) error {
	c.repoPath = path
	return nil
}

// Factory は CLI の依存関係を管理し、注入します。
type Factory struct {
	IOOut io.Writer
	IOErr io.Writer
	IOIn  io.Reader

	// Config は Config インスタンスを返す関数です（遅延ロードされます）。
	Config func() (Config, error)
}

// NewFactory は標準の入出力（os.Stdout/Stderr/Stdin）を使用して新しい Factory を作成します。
func NewFactory() *Factory {
	return &Factory{
		IOOut: os.Stdout,
		IOErr: os.Stderr,
		IOIn:  os.Stdin,
		Config: func() (Config, error) {
			// 実際のアプリケーションでは、設定ファイルから読み込みます。
			// 現時点では、メモリ内のダミー設定を返します。
			return &dummyConfig{repoPath: ""}, nil
		},
	}
}
