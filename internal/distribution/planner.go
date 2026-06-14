package distribution

import (
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// Planner は選択結果と現在状態から初回配布計画を構築します。
type Planner struct {
	fileSystem FileSystem
}

// NewPlanner は初回配布Plannerを返します。
func NewPlanner(fileSystem FileSystem) *Planner {
	return &Planner{fileSystem: fileSystem}
}

// Plan は選択結果と現在状態から初回配布計画を構築します。
//
//nolint:gocognit,cyclop // 選択の正規化、新旧の差分（Creates/Deletes）抽出、全対象の期待状態固定を一箇所で行います。
func (p *Planner) Plan(snapshot MapSnapshot, selection Selection) (Plan, error) {
	if err := validateSelection(selection); err != nil {
		return Plan{}, err
	}
	oldRecord, isManaged := snapshot.Workspaces[selection.WorkspaceRoot]

	oldSkillsMap := make(map[string]SkillRecord)
	if isManaged {
		for _, skill := range oldRecord.Skills {
			key := skill.Name + "\x00" + string(skill.Destination)
			oldSkillsMap[key] = skill
		}
	}

	destinations := append([]Destination(nil), selection.Destinations...)
	slices.Sort(destinations)
	skills := append([]SelectedSkill(nil), selection.Skills...)
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Name != skills[j].Name {
			return skills[i].Name < skills[j].Name
		}
		return skills[i].Source < skills[j].Source
	})

	workspace := WorkspaceRecord{
		WorkspaceRoot: selection.WorkspaceRoot,
		Project:       selection.Project,
		Destinations:  destinations,
	}
	plan := Plan{ExpectedRevision: snapshot.Revision, Workspace: workspace}

	// 追加または更新するSkillを算出
	for _, skill := range skills {
		sourceStates, err := p.inspectChain(skill.SourcePath, PathKindDirectory, false)
		if err != nil {
			return Plan{}, err
		}
		hash, err := p.fileSystem.HashSkill(skill.SourcePath)
		if err != nil {
			return Plan{}, newError("hash source", ErrIO, err)
		}
		for _, destination := range destinations {
			relative, err := destinationRelativePath(destination, skill.Name)
			if err != nil {
				return Plan{}, newError("plan destination", ErrStructure, err)
			}
			finalPath := filepath.Join(selection.WorkspaceRoot, relative)
			targetStates, err := p.inspectChain(finalPath, PathKindDirectory, true)
			if err != nil {
				return Plan{}, err
			}

			key := skill.Name + "\x00" + string(destination)
			oldSkill, isAlreadyManaged := oldSkillsMap[key]

			// 別プロジェクトへの切り替え時は、同名であっても新規/置換扱いとする
			if isAlreadyManaged && oldRecord.Project != selection.Project {
				isAlreadyManaged = false
			}

			// 既存の配布予定先ディレクトリの内容ハッシュを計算します（実在する場合のみ）
			targetExists := targetStates[len(targetStates)-1].Exists
			var currentTargetHash string
			var hashErr error
			if targetExists {
				currentTargetHash, hashErr = p.fileSystem.HashSkill(finalPath)
			}

			isConflict := false
			isLocalEdit := false

			if targetExists {
				if !isAlreadyManaged {
					// 管理対象ではないのに既に配布予定先にファイルが存在する場合は競合
					isConflict = true
				} else if hashErr != nil || currentTargetHash != oldSkill.Hash {
					// 管理対象だが、ディスク上のハッシュが前回配布時と異なる、またはハッシュ計算エラー（欠落含む）がある場合はローカル編集
					isLocalEdit = true
				}
			} else if isAlreadyManaged {
				// 管理対象のはずなのにディスク上に存在しない場合は欠落（ローカル編集扱い）
				isLocalEdit = true
			}

			if isAlreadyManaged && oldSkill.Hash == hash && !isLocalEdit {
				// 既に同一内容で配布済み、かつ変更もなく、ローカル編集もない場合はそのままWorkspaceRecordに維持
				plan.Workspace.Skills = append(plan.Workspace.Skills, oldSkill)
				continue
			}

			operation := CreateOperation{
				Name:             skill.Name,
				Source:           skill.Source,
				SourcePath:       skill.SourcePath,
				Destination:      destination,
				RelativePath:     filepath.ToSlash(relative),
				FinalPath:        finalPath,
				Hash:             hash,
				SourcePathStates: sourceStates,
				TargetPathStates: targetStates,
				IsConflict:       isConflict,
				IsLocalEdit:      isLocalEdit,
			}
			plan.Creates = append(plan.Creates, operation)
			plan.Workspace.Skills = append(plan.Workspace.Skills, SkillRecord{
				Name:         skill.Name,
				Source:       skill.Source,
				Destination:  destination,
				RelativePath: filepath.ToSlash(relative),
				Hash:         hash,
			})
		}
	}

	// 削除するSkillを算出
	if isManaged {
		for _, oldSkill := range oldRecord.Skills {
			retained := false
			if slices.Contains(selection.Destinations, oldSkill.Destination) {
				for _, s := range selection.Skills {
					if s.Name == oldSkill.Name {
						retained = true
						break
					}
				}
			}

			// プロジェクト切り替え時は古いプロジェクトのSkillはすべて削除対象とする
			if oldRecord.Project != selection.Project {
				retained = false
			}

			if !retained {
				finalPath := filepath.Join(selection.WorkspaceRoot, oldSkill.RelativePath)
				targetStates, err := p.inspectChain(finalPath, PathKindDirectory, true)
				if err != nil {
					return Plan{}, err
				}

				// 削除対象のローカル編集・欠落を検出します
				targetExists := targetStates[len(targetStates)-1].Exists
				isLocalEdit := false
				if targetExists {
					currentTargetHash, hashErr := p.fileSystem.HashSkill(finalPath)
					if hashErr != nil || currentTargetHash != oldSkill.Hash {
						isLocalEdit = true
					}
				} else {
					// すでにフォルダが存在しない（欠落）
					isLocalEdit = true
				}

				plan.Deletes = append(plan.Deletes, DeleteOperation{
					Name:             oldSkill.Name,
					Destination:      oldSkill.Destination,
					RelativePath:     oldSkill.RelativePath,
					FinalPath:        finalPath,
					TargetPathStates: targetStates,
					IsLocalEdit:      isLocalEdit,
				})
			}
		}
	}

	sort.Slice(plan.Creates, func(i, j int) bool {
		return plan.Creates[i].FinalPath < plan.Creates[j].FinalPath
	})
	sort.Slice(plan.Deletes, func(i, j int) bool {
		return plan.Deletes[i].FinalPath < plan.Deletes[j].FinalPath
	})
	sort.Slice(plan.Workspace.Skills, func(i, j int) bool {
		left, right := plan.Workspace.Skills[i], plan.Workspace.Skills[j]
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		if left.Destination != right.Destination {
			return left.Destination < right.Destination
		}
		return left.RelativePath < right.RelativePath
	})
	return plan, nil
}

func (p *Planner) inspectChain(path string, finalKind PathKind, allowMissing bool) ([]PathExpectation, error) {
	components := absolutePathComponents(path)
	expectations := make([]PathExpectation, 0, len(components))
	missing := false
	for index, component := range components {
		kind := PathKindDirectory
		if index == len(components)-1 {
			kind = finalKind
		}
		expectation, err := p.fileSystem.Inspect(component, kind, allowMissing || missing)
		if err != nil {
			return nil, newError("inspect path chain", ErrIO, err)
		}
		if !expectation.Exists {
			missing = true
		}
		expectations = append(expectations, expectation)
	}
	return expectations, nil
}

//nolint:gocognit,cyclop // 入力モデル全体の相互制約を一か所で検証します。
func validateSelection(selection Selection) error {
	if !filepath.IsAbs(selection.WorkspaceRoot) || filepath.Clean(selection.WorkspaceRoot) != selection.WorkspaceRoot {
		return newError("plan workspace", ErrUnsafePath, nil)
	}
	if !validName(selection.Project) {
		return newError("plan project", ErrStructure, nil)
	}
	seenSkills := make(map[string]struct{}, len(selection.Skills))
	for _, skill := range selection.Skills {
		if !validName(skill.Name) || !filepath.IsAbs(skill.SourcePath) {
			return newError("plan skill", ErrStructure, nil)
		}
		if skill.Source != SkillSourceProject && skill.Source != SkillSourceCommon {
			return newError("plan skill", ErrStructure, nil)
		}
		if _, exists := seenSkills[skill.Name]; exists {
			return newError("plan skill", ErrStructure, nil)
		}
		seenSkills[skill.Name] = struct{}{}
	}
	seenDestinations := make(map[Destination]struct{}, len(selection.Destinations))
	for _, destination := range selection.Destinations {
		if destination != DestinationCodex && destination != DestinationClaude {
			return newError("plan destination", ErrStructure, nil)
		}
		if _, exists := seenDestinations[destination]; exists {
			return newError("plan destination", ErrStructure, nil)
		}
		seenDestinations[destination] = struct{}{}
	}
	if len(selection.Skills) > 0 && len(selection.Destinations) == 0 {
		return newError("plan destination", ErrStructure, nil)
	}
	return nil
}

func validName(name string) bool {
	return name != "" && name != "." && name != ".." &&
		!strings.ContainsRune(name, filepath.Separator) &&
		!strings.Contains(name, "/") && !strings.Contains(name, "\\")
}

func absolutePathComponents(path string) []string {
	cleaned := filepath.Clean(path)
	var reversed []string
	for current := cleaned; ; current = filepath.Dir(current) {
		reversed = append(reversed, current)
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	components := make([]string, len(reversed))
	for index := range reversed {
		components[len(reversed)-1-index] = reversed[index]
	}
	return components
}
