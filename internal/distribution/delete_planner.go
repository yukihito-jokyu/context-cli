package distribution

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
)

// PlanDelete は削除対象のSkill名と現在状態から削除計画を構築します。
//
//nolint:gocognit,cyclop // 削除対象の検証、残るSkillの抽出、削除対象の期待状態固定を一箇所で行います。
func (p *Planner) PlanDelete(snapshot MapSnapshot, workspaceRoot string, skillNames []string) (Plan, error) {
	// 1. Workspaceが管理されているか検証
	oldRecord, isManaged := snapshot.Workspaces[workspaceRoot]
	if !isManaged {
		return Plan{}, newError("plan delete", ErrUnmanagedWorkspace, nil)
	}

	// 2. 引数の重複排除
	uniqueSkillNames := make([]string, 0, len(skillNames))
	seenInput := make(map[string]struct{})
	for _, name := range skillNames {
		if _, ok := seenInput[name]; !ok {
			seenInput[name] = struct{}{}
			uniqueSkillNames = append(uniqueSkillNames, name)
		}
	}

	// 3. 記録されている既存Skillのマップ作成
	managedSkills := make(map[string]bool)
	for _, s := range oldRecord.Skills {
		managedSkills[s.Name] = true
	}

	// 4. 引数のSkillがすべて存在するか検証
	for _, name := range uniqueSkillNames {
		if !managedSkills[name] {
			return Plan{}, newError("plan delete", ErrPrecondition, fmt.Errorf("skill %q is not distributed in this workspace: %w", name, ErrPrecondition))
		}
	}

	// 5. 削除対象の分類用マップ
	toDelete := make(map[string]bool)
	for _, name := range uniqueSkillNames {
		toDelete[name] = true
	}

	workspace := WorkspaceRecord{
		WorkspaceRoot: oldRecord.WorkspaceRoot,
		Project:       oldRecord.Project,
	}

	var deletes []DeleteOperation
	var retainedSkills []SkillRecord
	retainedDests := make(map[Destination]bool)

	for _, oldSkill := range oldRecord.Skills {
		if toDelete[oldSkill.Name] {
			// 削除対象のSkillの場合、DeleteOperationを生成
			finalPath := filepath.Join(workspaceRoot, oldSkill.RelativePath)
			targetStates, err := p.inspectChain(finalPath, PathKindDirectory, true)
			if err != nil {
				return Plan{}, err
			}

			targetExists := targetStates[len(targetStates)-1].Exists
			isLocalEdit := false
			if targetExists {
				currentTargetHash, hashErr := p.fileSystem.HashSkill(finalPath)
				if hashErr != nil || currentTargetHash != oldSkill.Hash {
					isLocalEdit = true
				}
			} else {
				// 欠落している場合もローカル編集扱い
				isLocalEdit = true
			}

			deletes = append(deletes, DeleteOperation{
				Name:             oldSkill.Name,
				Destination:      oldSkill.Destination,
				RelativePath:     oldSkill.RelativePath,
				FinalPath:        finalPath,
				TargetPathStates: targetStates,
				IsLocalEdit:      isLocalEdit,
			})
		} else {
			// 維持するSkillの場合
			retainedSkills = append(retainedSkills, oldSkill)
			retainedDests[oldSkill.Destination] = true
		}
	}

	// 6. 維持するSkillの情報をWorkspaceRecordに設定
	if len(retainedSkills) > 0 {
		workspace.Skills = retainedSkills
		var destinations []Destination
		for dest := range retainedDests {
			destinations = append(destinations, dest)
		}
		slices.Sort(destinations)
		workspace.Destinations = destinations
	} else {
		workspace.Skills = nil
		workspace.Destinations = nil
	}

	// 7. 決定論的な順序にソート
	sort.Slice(deletes, func(i, j int) bool {
		return deletes[i].FinalPath < deletes[j].FinalPath
	})
	if workspace.Skills != nil {
		sort.Slice(workspace.Skills, func(i, j int) bool {
			left, right := workspace.Skills[i], workspace.Skills[j]
			if left.Name != right.Name {
				return left.Name < right.Name
			}
			if left.Destination != right.Destination {
				return left.Destination < right.Destination
			}
			return left.RelativePath < right.RelativePath
		})
	}

	plan := Plan{
		ExpectedRevision: snapshot.Revision,
		Workspace:        workspace,
		Creates:          nil,
		Deletes:          deletes,
	}

	return plan, nil
}
