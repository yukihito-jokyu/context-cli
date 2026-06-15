package cmd

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/yukihito-jokyu/context-cli/internal/distribution"
	"github.com/yukihito-jokyu/context-cli/internal/skillcatalog"
)

// SkillKind は選択するSkill候補の種別を表します。
type SkillKind string

const (
	// SkillKindProject はプロジェクト固有Skillを表します。
	SkillKindProject SkillKind = "project-skills"
	// SkillKindCommon は共通Skillを表します。
	SkillKindCommon SkillKind = "common-skills"
)

// Prompt はAddコマンドの対話操作を表します。
//
//nolint:interfacebloat // 対話コマンドで必要なすべての選択ダイアログを1つのインターフェースにまとめます。
type Prompt interface {
	SelectProject(candidates []skillcatalog.Candidate, defaultName string) (skillcatalog.Candidate, error)
	SelectSkills(kind SkillKind, candidates []skillcatalog.Candidate, defaultNames []string) ([]skillcatalog.Candidate, error)
	ConfirmCommonSkills(defaultConfirmed bool) (bool, error)
	SelectDestinations(defaultDestinations []distribution.Destination) ([]distribution.Destination, error)
	ConfirmOverwrite(conflicts []string, localEdits []string) (bool, error)
	ConfirmSync(updates []string, deletes []string) (bool, error)
	SelectSkillsToDelete(candidates []string) ([]string, error)
}

type huhPrompt struct {
	input  io.Reader
	output io.Writer
	run    func(*huh.Form) error
}

func newHuhPrompt(input io.Reader, output io.Writer) Prompt {
	return &huhPrompt{
		input:  input,
		output: output,
		run: func(form *huh.Form) error {
			return form.Run()
		},
	}
}

func (p *huhPrompt) SelectProject(candidates []skillcatalog.Candidate, defaultName string) (skillcatalog.Candidate, error) {
	var selected skillcatalog.Candidate
	options := make([]huh.Option[skillcatalog.Candidate], len(candidates))
	for i, candidate := range candidates {
		options[i] = huh.NewOption(candidate.Name, candidate)
		if candidate.Name == defaultName {
			selected = candidate
		}
	}
	err := p.runField(huh.NewSelect[skillcatalog.Candidate]().
		Title("プロジェクトを選択してください").
		Options(options...).
		Value(&selected))
	return selected, err
}

func (p *huhPrompt) SelectSkills(kind SkillKind, candidates []skillcatalog.Candidate, defaultNames []string) ([]skillcatalog.Candidate, error) {
	selected := []skillcatalog.Candidate{}
	options := make([]huh.Option[skillcatalog.Candidate], len(candidates))
	for i, candidate := range candidates {
		options[i] = huh.NewOption(candidate.Name, candidate)
		if slices.Contains(defaultNames, candidate.Name) {
			selected = append(selected, candidate)
		}
	}
	title := "プロジェクト固有Skillを選択してください"
	if kind == SkillKindCommon {
		title = "共通Skillを選択してください"
	}
	err := p.runField(huh.NewMultiSelect[skillcatalog.Candidate]().
		Title(title).
		Options(options...).
		Value(&selected))
	return selected, err
}

func (p *huhPrompt) ConfirmCommonSkills(defaultConfirmed bool) (bool, error) {
	selected := defaultConfirmed
	field := newCommonSkillsConfirm(&selected)
	err := p.runField(field)
	return selected, err
}

func newCommonSkillsConfirm(selected *bool) *huh.Confirm {
	return huh.NewConfirm().
		Title("共通Skillを追加しますか?").
		Affirmative("はい").
		Negative("いいえ").
		Value(selected)
}

func (p *huhPrompt) SelectDestinations(defaultDestinations []distribution.Destination) ([]distribution.Destination, error) {
	selected := append([]distribution.Destination(nil), defaultDestinations...)
	err := p.runField(huh.NewMultiSelect[distribution.Destination]().
		Title("配布先を選択してください").
		Options(
			huh.NewOption("Codex", distribution.DestinationCodex),
			huh.NewOption("Claude", distribution.DestinationClaude),
		).
		Validate(validateDestinations).
		Value(&selected))
	return selected, err
}

func validateDestinations(value []distribution.Destination) error {
	if len(value) == 0 {
		return ErrDestinationRequired
	}
	return nil
}

func (p *huhPrompt) runField(field huh.Field) error {
	form := huh.NewForm(huh.NewGroup(field)).
		WithInput(p.input).
		WithOutput(p.output)
	err := p.run(form)
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return huh.ErrUserAborted
		}
		return fmt.Errorf("%w: %w", ErrPrompt, err)
	}
	return nil
}

func (p *huhPrompt) ConfirmOverwrite(conflicts []string, localEdits []string) (bool, error) {
	var sb strings.Builder
	if len(conflicts) > 0 {
		sb.WriteString("以下の未管理ファイルが衝突しています（上書きされます）:\n")
		for _, path := range conflicts {
			fmt.Fprintf(&sb, "  - %s\n", path)
		}
	}
	if len(localEdits) > 0 {
		sb.WriteString("以下のファイルがローカルで編集されています（上書き・削除されます）:\n")
		for _, path := range localEdits {
			fmt.Fprintf(&sb, "  - %s\n", path)
		}
	}
	sb.WriteString("\nこれらの変更を承認し、処理を続行しますか？")

	selected := false
	field := huh.NewConfirm().
		Title(sb.String()).
		Affirmative("はい").
		Negative("いいえ").
		Value(&selected)
	err := p.runField(field)
	return selected, err
}

func (p *huhPrompt) ConfirmSync(updates []string, deletes []string) (bool, error) {
	var sb strings.Builder
	if len(updates) > 0 {
		sb.WriteString("以下のファイルを更新します:\n")
		for _, path := range updates {
			fmt.Fprintf(&sb, "  - %s\n", path)
		}
	}
	if len(deletes) > 0 {
		sb.WriteString("以下のファイルを削除します:\n")
		for _, path := range deletes {
			fmt.Fprintf(&sb, "  - %s\n", path)
		}
	}
	sb.WriteString("\nこれらの変更を承認し、同期を実行しますか？")

	selected := false
	field := huh.NewConfirm().
		Title(sb.String()).
		Affirmative("はい").
		Negative("いいえ").
		Value(&selected)
	err := p.runField(field)
	return selected, err
}

func (p *huhPrompt) SelectSkillsToDelete(candidates []string) ([]string, error) {
	selected := []string{}
	options := make([]huh.Option[string], len(candidates))
	for i, name := range candidates {
		options[i] = huh.NewOption(name, name)
	}
	err := p.runField(huh.NewMultiSelect[string]().
		Title("削除するSkillを選択してください").
		Options(options...).
		Value(&selected))
	return selected, err
}
