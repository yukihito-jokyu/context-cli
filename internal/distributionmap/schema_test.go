package distributionmap

import (
	"errors"
	"testing"

	"github.com/yukihito-jokyu/context-cli/internal/distribution"
)

func TestDecodeRejectsUnknownFieldsAndInvalidRecords(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "未知フィールド", data: "schema_version: 1\nunknown: true\nworkspaces: {}\n"},
		{name: "未対応バージョン", data: "schema_version: 2\nworkspaces: {}\n"},
		{name: "相対Workspace", data: "schema_version: 1\nworkspaces:\n  relative:\n    project: p\n    destinations: [codex]\n    skills: []\n"},
		{name: "不正ハッシュ", data: "schema_version: 1\nworkspaces:\n  /workspace:\n    project: p\n    destinations: [codex]\n    skills:\n      - name: s\n        source: project\n        destination: codex\n        relative_path: .codex/skills/s\n        hash: invalid\n"},
		{name: "重複Skill", data: "schema_version: 1\nworkspaces:\n  /workspace:\n    project: p\n    destinations: [codex]\n    skills:\n      - &skill\n        name: s\n        source: project\n        destination: codex\n        relative_path: .codex/skills/s\n        hash: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n      - *skill\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decode([]byte(tt.data))
			if !errors.Is(err, ErrSchema) {
				t.Fatalf("decode() error = %v, want ErrSchema", err)
			}
		})
	}
}

func TestSchemaAllowsSameSkillForCodexAndClaude(t *testing.T) {
	record := distribution.WorkspaceRecord{
		WorkspaceRoot: "/workspace",
		Project:       "project",
		Destinations:  []distribution.Destination{distribution.DestinationCodex, distribution.DestinationClaude},
		Skills: []distribution.SkillRecord{
			{Name: "skill", Source: distribution.SkillSourceProject, Destination: distribution.DestinationCodex, RelativePath: ".codex/skills/skill", Hash: hashA},
			{Name: "skill", Source: distribution.SkillSourceProject, Destination: distribution.DestinationClaude, RelativePath: ".claude/skills/skill", Hash: hashA},
		},
	}
	data, err := encode(map[string]distribution.WorkspaceRecord{record.WorkspaceRoot: record})
	if err != nil {
		t.Fatalf("encode() error = %v", err)
	}
	decoded, err := decode(data)
	if err != nil {
		t.Fatalf("decode() error = %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("len(decoded) = %d, want 1", len(decoded))
	}
}

const hashA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
