package distribution

import (
	"path/filepath"
	"slices"
	"sort"
)

// SyncPlanner は解決済み供給元とWorkspace記録から同期計画を構築します。
type SyncPlanner struct {
	fileSystem FileSystem
}

// NewSyncPlanner は同期専用Plannerを返します。
func NewSyncPlanner(fileSystem FileSystem) *SyncPlanner {
	return &SyncPlanner{fileSystem: fileSystem}
}

// syncContext は同期計画構築中の状態をスレッド化するための作業コンテキストです。
type syncContext struct {
	plan      *SyncPlan
	workspace *WorkspaceRecord
	input     SyncInput
}

// syncTarget は1配布先の解決済み情報をまとめます。
type syncTarget struct {
	destination  Destination
	relative     string
	finalPath    string
	recordedHash string
}

// Plan は同期入力と解決済み供給元から更新・削除・維持の同期計画を構築します。
//
// 供給元不変かつ配布先のみ変更された場合も更新とします。
// 配布先の親経路または末端がシンボリックリンクの場合は安全性エラーで停止します。
// 全操作と確認対象を決定的な相対パス順へ並べます。
func (p *SyncPlanner) Plan(snapshot MapSnapshot, input SyncInput, sources []ResolvedSource) (SyncPlan, error) {
	if err := validateSyncInput(input); err != nil {
		return SyncPlan{}, err
	}
	resolved := normalizeResolvedSources(sources)
	destinations := append([]Destination(nil), input.Destinations...)
	slices.Sort(destinations)

	plan := SyncPlan{
		ExpectedRevision: snapshot.Revision,
		WorkspaceRoot:    input.WorkspaceRoot,
		Project:          input.Project,
		Destinations:     destinations,
		ResolvedSources:  append([]ResolvedSource(nil), resolved...),
	}
	workspace := WorkspaceRecord{
		WorkspaceRoot: input.WorkspaceRoot,
		Project:       input.Project,
		Destinations:  destinations,
	}
	context := syncContext{plan: &plan, workspace: &workspace, input: input}

	for _, source := range resolved {
		recordedDestinations := destinationsForSource(input.Skills, source.Name)
		if len(recordedDestinations) == 0 {
			// 記録にないSkill名は候補列挙しない本仕様では発生し得ないが、安全のため飛ばす。
			continue
		}
		if err := p.planSource(&context, source, recordedDestinations); err != nil {
			return SyncPlan{}, err
		}
	}

	finalizeSyncPlan(&plan, &workspace)
	return plan, nil
}

// planSource は供給元状態に応じて更新・維持または削除計画へ振り分けます。
func (p *SyncPlanner) planSource(
	context *syncContext,
	source ResolvedSource,
	destinations []Destination,
) error {
	switch source.State {
	case SourceStateActive:
		return p.planActiveSource(context, source, destinations)
	case SourceStateMissing, SourceStateDisabled:
		return p.planRemovedSource(context, source, destinations)
	default:
		return newError("sync source state", ErrStructure, nil)
	}
}

func (p *SyncPlanner) planActiveSource(
	context *syncContext,
	source ResolvedSource,
	destinations []Destination,
) error {
	sourceStates := source.SourcePathStates
	if len(sourceStates) == 0 {
		states, err := p.inspectChain(source.Path, PathKindDirectory, false)
		if err != nil {
			return err
		}
		sourceStates = states
	}
	sourceHash := source.Hash
	if sourceHash == "" {
		hash, err := p.fileSystem.HashSkill(source.Path)
		if err != nil {
			return newError("sync hash source", ErrIO, err)
		}
		sourceHash = hash
	}

	for _, destination := range destinations {
		relative, err := destinationRelativePath(destination, source.Name)
		if err != nil {
			return newError("sync destination", ErrStructure, err)
		}
		target := syncTarget{
			destination:  destination,
			relative:     relative,
			finalPath:    filepath.Join(context.input.WorkspaceRoot, relative),
			recordedHash: recordedHashFor(context.input.Skills, source.Name, destination),
		}

		operation, err := p.classifyDestination(source, sourceHash, sourceStates, target)
		if err != nil {
			return err
		}
		if err := context.recordActiveOperation(source, sourceHash, operation, target); err != nil {
			return err
		}
	}
	return nil
}

func (p *SyncPlanner) planRemovedSource(
	context *syncContext,
	source ResolvedSource,
	destinations []Destination,
) error {
	for _, destination := range destinations {
		relative, err := destinationRelativePath(destination, source.Name)
		if err != nil {
			return newError("sync destination", ErrStructure, err)
		}
		target := syncTarget{
			destination:  destination,
			relative:     relative,
			finalPath:    filepath.Join(context.input.WorkspaceRoot, relative),
			recordedHash: recordedHashFor(context.input.Skills, source.Name, destination),
		}

		operation, err := p.classifyRemoval(source, target)
		if err != nil {
			return err
		}
		context.plan.Deletes = append(context.plan.Deletes, operation)
		if operation.IsLocalEdit {
			context.plan.LocalChanges = append(context.plan.LocalChanges, operation)
		}
		// 消失・無効化したSkillは更新後Workspace記録へ含めない（削除扱い）。
	}
	return nil
}

// recordActiveOperation は有効供給元の判定結果を計画と更新後記録へ反映します。
func (context *syncContext) recordActiveOperation(
	source ResolvedSource,
	sourceHash string,
	operation SyncOperation,
	target syncTarget,
) error {
	switch operation.Kind {
	case SyncOperationKeep:
		context.plan.Keeps = append(context.plan.Keeps, operation)
		context.workspace.Skills = append(context.workspace.Skills, SkillRecord{
			Name:         source.Name,
			Source:       source.Source,
			Destination:  target.destination,
			RelativePath: filepath.ToSlash(target.relative),
			Hash:         target.recordedHash,
		})
	case SyncOperationUpdate:
		context.plan.Updates = append(context.plan.Updates, operation)
		if operation.IsLocalEdit {
			context.plan.LocalChanges = append(context.plan.LocalChanges, operation)
		}
		context.workspace.Skills = append(context.workspace.Skills, SkillRecord{
			Name:         source.Name,
			Source:       source.Source,
			Destination:  target.destination,
			RelativePath: filepath.ToSlash(target.relative),
			Hash:         sourceHash,
		})
	case SyncOperationDelete:
		// 有効供給元の判定結果に削除は含まれない。到達時は計画構築側の不整合。
		return newError("sync operation kind", ErrStructure, nil)
	}
	return nil
}

// classifyDestination は有効供給元について配布先の更新・維持・ローカル変更を判定します。
//
// 親経路が実ディレクトリ以外（シンボリックリンク含む）や末端がシンボリックリンクの場合は
// inspectChainがErrSymlink/ErrFileTypeを返すため、ここで安全性エラーとして停止します。
func (p *SyncPlanner) classifyDestination(
	source ResolvedSource,
	sourceHash string,
	sourceStates []PathExpectation,
	target syncTarget,
) (SyncOperation, error) {
	targetStates, err := p.inspectChain(target.finalPath, PathKindAny, true)
	if err != nil {
		return SyncOperation{}, err
	}
	leaf := targetStates[len(targetStates)-1]

	currentHash := ""
	isLocalEdit := false
	if leaf.Exists {
		if leaf.Kind != PathKindDirectory {
			// 末端がディレクトリ以外（通常ファイル・FIFO・ソケット・デバイス等）へ変化。
			// シンボリックリンク以外は承認可能なローカル変更。
			isLocalEdit = true
		} else {
			hash, hashErr := p.fileSystem.HashSkill(target.finalPath)
			if hashErr != nil {
				return SyncOperation{}, newError("sync hash target", ErrIO, hashErr)
			}
			currentHash = hash
			if hash != target.recordedHash {
				isLocalEdit = true
			}
		}
	} else {
		// 欠落は承認可能なローカル変更。
		isLocalEdit = true
	}

	operation := SyncOperation{
		Name:             source.Name,
		Source:           source.Source,
		Destination:      target.destination,
		RelativePath:     filepath.ToSlash(target.relative),
		FinalPath:        target.finalPath,
		RecordedHash:     target.recordedHash,
		CurrentHash:      currentHash,
		IsLocalEdit:      isLocalEdit,
		SourcePath:       source.Path,
		SourceHash:       sourceHash,
		SourcePathStates: sourceStates,
		TargetPathStates: targetStates,
	}

	// 供給元不変かつ配布先も記録ハッシュと一致し、末端が安全なディレクトリの場合は維持。
	if sourceHash == target.recordedHash && !isLocalEdit && leaf.Exists && leaf.Kind == PathKindDirectory {
		operation.Kind = SyncOperationKeep
		return operation, nil
	}

	operation.Kind = SyncOperationUpdate
	return operation, nil
}

func (p *SyncPlanner) classifyRemoval(source ResolvedSource, target syncTarget) (SyncOperation, error) {
	targetStates, err := p.inspectChain(target.finalPath, PathKindAny, true)
	if err != nil {
		return SyncOperation{}, err
	}
	leaf := targetStates[len(targetStates)-1]

	isLocalEdit := false
	currentHash := ""
	if leaf.Exists {
		if leaf.Kind != PathKindDirectory {
			isLocalEdit = true
		} else {
			hash, hashErr := p.fileSystem.HashSkill(target.finalPath)
			if hashErr != nil {
				return SyncOperation{}, newError("sync hash removal", ErrIO, hashErr)
			}
			currentHash = hash
			if hash != target.recordedHash {
				isLocalEdit = true
			}
		}
	} else {
		isLocalEdit = true
	}

	return SyncOperation{
		Kind:             SyncOperationDelete,
		Name:             source.Name,
		Source:           source.Source,
		Destination:      target.destination,
		RelativePath:     filepath.ToSlash(target.relative),
		FinalPath:        target.finalPath,
		RecordedHash:     target.recordedHash,
		CurrentHash:      currentHash,
		IsLocalEdit:      isLocalEdit,
		TargetPathStates: targetStates,
	}, nil
}

// ToPlan は同期計画をExecutor実行用のPlanへ変換します。
//
// 更新対象はCreateOperation（SourcePathStatesに供給元期待状態）、
// 削除対象はDeleteOperationへ変換します。
// 計画時に固定した期待状態をそのまま使うため、ファイルシステムへ再アクセスしません。
func (plan SyncPlan) ToPlan() (Plan, error) {
	result := Plan{
		ExpectedRevision: plan.ExpectedRevision,
		Workspace:        plan.UpdatedWorkspace,
		IsSync:           true,
	}
	for _, operation := range plan.Updates {
		result.Creates = append(result.Creates, CreateOperation{
			Name:             operation.Name,
			Source:           operation.Source,
			SourcePath:       operation.SourcePath,
			Destination:      operation.Destination,
			RelativePath:     operation.RelativePath,
			FinalPath:        operation.FinalPath,
			Hash:             operation.SourceHash,
			SourcePathStates: operation.SourcePathStates,
			TargetPathStates: operation.TargetPathStates,
			IsConflict:       false,
			IsLocalEdit:      operation.IsLocalEdit,
		})
	}
	for _, operation := range plan.Deletes {
		result.Deletes = append(result.Deletes, DeleteOperation{
			Name:             operation.Name,
			Destination:      operation.Destination,
			RelativePath:     operation.RelativePath,
			FinalPath:        operation.FinalPath,
			TargetPathStates: operation.TargetPathStates,
			IsLocalEdit:      operation.IsLocalEdit,
		})
	}
	sort.Slice(result.Creates, func(i, j int) bool {
		return result.Creates[i].FinalPath < result.Creates[j].FinalPath
	})
	sort.Slice(result.Deletes, func(i, j int) bool {
		return result.Deletes[i].FinalPath < result.Deletes[j].FinalPath
	})
	return result, nil
}

func (p *SyncPlanner) inspectChain(path string, finalKind PathKind, allowMissing bool) ([]PathExpectation, error) {
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
			return nil, newError("sync inspect path chain", ErrIO, err)
		}
		if !expectation.Exists {
			missing = true
		}
		expectations = append(expectations, expectation)
	}
	return expectations, nil
}

func validateSyncInput(input SyncInput) error {
	if err := validateSyncWorkspace(input); err != nil {
		return err
	}
	if !validName(input.Project) {
		return newError("sync project", ErrStructure, nil)
	}
	if err := validateSyncDestinations(input.Destinations); err != nil {
		return err
	}
	return validateSyncSkills(input.Skills)
}

func validateSyncWorkspace(input SyncInput) error {
	if !filepath.IsAbs(input.WorkspaceRoot) || filepath.Clean(input.WorkspaceRoot) != input.WorkspaceRoot {
		return newError("sync workspace", ErrUnsafePath, nil)
	}
	return nil
}

func validateSyncDestinations(destinations []Destination) error {
	seen := make(map[Destination]struct{}, len(destinations))
	for _, destination := range destinations {
		if destination != DestinationCodex && destination != DestinationClaude {
			return newError("sync destination", ErrStructure, nil)
		}
		if _, exists := seen[destination]; exists {
			return newError("sync destination", ErrStructure, nil)
		}
		seen[destination] = struct{}{}
	}
	return nil
}

func validateSyncSkills(skills []RecordedSkill) error {
	seen := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		if !validName(skill.Name) {
			return newError("sync skill", ErrStructure, nil)
		}
		if skill.Source != SkillSourceProject && skill.Source != SkillSourceCommon {
			return newError("sync skill", ErrStructure, nil)
		}
		if skill.Destination != DestinationCodex && skill.Destination != DestinationClaude {
			return newError("sync skill", ErrStructure, nil)
		}
		key := string(skill.Source) + "\x00" + skill.Name + "\x00" + string(skill.Destination)
		if _, exists := seen[key]; exists {
			return newError("sync skill", ErrStructure, nil)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func normalizeResolvedSources(sources []ResolvedSource) []ResolvedSource {
	resolved := append([]ResolvedSource(nil), sources...)
	sort.Slice(resolved, func(i, j int) bool {
		if resolved[i].Source != resolved[j].Source {
			return resolved[i].Source < resolved[j].Source
		}
		return resolved[i].Name < resolved[j].Name
	})
	return resolved
}

func finalizeSyncPlan(plan *SyncPlan, workspace *WorkspaceRecord) {
	plan.UpdatedWorkspace = *workspace
	sortOperationsByNameDestination(plan.Updates)
	sortOperationsByNameDestination(plan.Deletes)
	sortOperationsByNameDestination(plan.Keeps)
	sortOperationsByNameDestination(plan.LocalChanges)
	sort.Slice(plan.UpdatedWorkspace.Skills, func(i, j int) bool {
		left, right := plan.UpdatedWorkspace.Skills[i], plan.UpdatedWorkspace.Skills[j]
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.Destination < right.Destination
	})
}

func sortOperationsByNameDestination(operations []SyncOperation) {
	sort.Slice(operations, func(i, j int) bool {
		if operations[i].Name != operations[j].Name {
			return operations[i].Name < operations[j].Name
		}
		return operations[i].Destination < operations[j].Destination
	})
}

func lookupRecordedSkill(skills []RecordedSkill, name string, destination Destination) *RecordedSkill {
	for index := range skills {
		if skills[index].Name == name && skills[index].Destination == destination {
			return &skills[index]
		}
	}
	return nil
}

func recordedHashFor(skills []RecordedSkill, name string, destination Destination) string {
	recorded := lookupRecordedSkill(skills, name, destination)
	if recorded == nil {
		return ""
	}
	return recorded.RecordedHash
}

// destinationsForSource は記録済みSkillのうち指定Skill名の配布先を名前順で返します。
func destinationsForSource(skills []RecordedSkill, name string) []Destination {
	seen := make(map[Destination]struct{})
	var destinations []Destination
	for _, skill := range skills {
		if skill.Name != name {
			continue
		}
		if _, exists := seen[skill.Destination]; exists {
			continue
		}
		seen[skill.Destination] = struct{}{}
		destinations = append(destinations, skill.Destination)
	}
	slices.Sort(destinations)
	return destinations
}
