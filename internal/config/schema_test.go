package config

import (
	"errors"
	"strings"
	"testing"
)

func TestDecodeSchema(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPath   string
		wantErr    error
		wantSecret string
	}{
		{
			name:     "有効な設定を読み込む",
			input:    "schema_version: 1\ncontext_repository: /tmp/context\n",
			wantPath: "/tmp/context",
		},
		{
			name:     "旧形式の設定を読み込む",
			input:    "version: 1\nrepository_path: /tmp/legacy-context\n",
			wantPath: "/tmp/legacy-context",
		},
		{
			name:    "旧形式の未知フィールドを拒否する",
			input:   "version: 1\nrepository_path: /tmp/context\nunknown: true\n",
			wantErr: ErrFormat,
		},
		{
			name:    "旧形式の未対応バージョンを拒否する",
			input:   "version: 2\nrepository_path: /tmp/context\n",
			wantErr: ErrSchema,
		},
		{
			name:    "新旧形式の混在を拒否する",
			input:   "schema_version: 1\ncontext_repository: /tmp/context\nversion: 1\nrepository_path: /tmp/legacy\n",
			wantErr: ErrFormat,
		},
		{
			name:    "不正なYAMLを拒否する",
			input:   "schema_version: [\n",
			wantErr: ErrFormat,
		},
		{
			name:    "未知フィールドを拒否する",
			input:   "schema_version: 1\ncontext_repository: /tmp/context\nunknown: true\n",
			wantErr: ErrFormat,
		},
		{
			name:    "複数ドキュメントを拒否する",
			input:   "schema_version: 1\ncontext_repository: /tmp/context\n---\nschema_version: 1\ncontext_repository: /tmp/other\n",
			wantErr: ErrFormat,
		},
		{
			name:    "未対応バージョンを拒否する",
			input:   "schema_version: 2\ncontext_repository: /tmp/context\n",
			wantErr: ErrSchema,
		},
		{
			name:    "空のRepositoryを拒否する",
			input:   "schema_version: 1\ncontext_repository: \"\"\n",
			wantErr: ErrSchema,
		},
		{
			name:       "相対Repositoryを拒否して内容を公開しない",
			input:      "schema_version: 1\ncontext_repository: secret/relative/path\n",
			wantErr:    ErrSchema,
			wantSecret: "secret/relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := decodeSchema([]byte(tt.input))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("decodeSchema() error = %v, want errors.Is(_, %v)", err, tt.wantErr)
			}
			if schema.ContextRepository != tt.wantPath {
				t.Errorf("ContextRepository = %q, want %q", schema.ContextRepository, tt.wantPath)
			}
			if tt.wantSecret != "" && strings.Contains(err.Error(), tt.wantSecret) {
				t.Errorf("エラー文字列に設定内容が含まれています: %q", err)
			}
		})
	}
}

func TestEncodeSchema(t *testing.T) {
	data, err := encodeSchema("/tmp/context")
	if err != nil {
		t.Fatalf("encodeSchema() error = %v", err)
	}
	if string(data) != "schema_version: 1\ncontext_repository: /tmp/context\n" {
		t.Errorf("encodeSchema() = %q", data)
	}

	_, err = encodeSchema("relative/context")
	if !errors.Is(err, ErrSchema) {
		t.Fatalf("encodeSchema() error = %v, want ErrSchema", err)
	}
}
