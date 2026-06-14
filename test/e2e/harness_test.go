package e2e_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/yukihito-jokyu/context-cli/pkg/cmd"
)

var (
	errMemoryConfigConflict = errors.New("設定値が変更されています")
	errTestFileLocation     = errors.New("テストファイルの位置を取得できません")
)

const binaryBuildTimeout = 30 * time.Second

//nolint:gochecknoglobals // TestMainと各テストで遅延ビルド結果を共有する。
var binaryHarness struct {
	once       sync.Once
	binaryPath string
	tempDir    string
	err        error
}

type harness struct {
	factory *cmd.Factory
	config  *memoryConfig
	output  *bytes.Buffer
	errOut  *bytes.Buffer
}

func newHarness(initialRepository, input string) *harness {
	output := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	config := &memoryConfig{repository: initialRepository}
	factory := cmd.NewFactory()
	factory.IOOut = output
	factory.IOErr = errOut
	factory.IOIn = bytes.NewBufferString(input)
	factory.Config = func() (cmd.Config, error) {
		return config, nil
	}
	return &harness{factory: factory, config: config, output: output, errOut: errOut}
}

func (h *harness) execute(args ...string) error {
	root := cmd.NewCmdRoot(h.factory)
	root.SetArgs(args)
	root.SetOut(h.output)
	root.SetErr(h.factory.IOErr)
	root.SilenceUsage = true
	root.SilenceErrors = true
	if err := root.Execute(); err != nil {
		return fmt.Errorf("CLI execution failed: %w", err)
	}
	return nil
}

type memoryConfig struct {
	repository string
	setCalls   int
}

func (c *memoryConfig) GetContextRepository() string {
	return c.repository
}

func (c *memoryConfig) SetContextRepository(expected, path string) error {
	if c.repository != expected {
		return errMemoryConfigConflict
	}
	c.setCalls++
	c.repository = path
	return nil
}

func createRepositoryFixture(t *testing.T) string {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("一時ディレクトリの実体パスを取得できません: %v", err)
	}
	repository := filepath.Join(base, "context")
	for _, path := range []string{
		filepath.Join(repository, "projects"),
		filepath.Join(repository, "utils", "skills"),
	} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatalf("fixtureを作成できません: %v", err)
		}
	}
	return repository
}

func changeWorkingDirectory(t *testing.T, path string) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("作業ディレクトリを取得できません: %v", err)
	}
	if err := os.Chdir(path); err != nil {
		t.Fatalf("作業ディレクトリを変更できません: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("作業ディレクトリを復元できません: %v", err)
		}
	})
}

type processResult struct {
	exitCode int
	stdout   string
	stderr   string
}

type processRequest struct {
	xdgConfigHome    string
	workingDirectory string
	stdin            string
	args             []string
}

type terminalStep struct {
	waitFor string
	input   string
}

func TestMain(m *testing.M) {
	code := m.Run()
	if binaryHarness.tempDir != "" {
		if err := os.RemoveAll(binaryHarness.tempDir); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "E2E用一時バイナリの削除に失敗しました")
			if code == 0 {
				code = 1
			}
		}
	}
	os.Exit(code)
}

func contextBinary(t *testing.T) string {
	t.Helper()
	binaryHarness.once.Do(buildContextBinary)
	if binaryHarness.err != nil {
		t.Fatalf("実バイナリをビルドできません: %v", binaryHarness.err)
	}
	return binaryHarness.binaryPath
}

func buildContextBinary() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		binaryHarness.err = errTestFileLocation
		return
	}
	repositoryRoot, err := filepath.EvalSymlinks(filepath.Join(filepath.Dir(filename), "..", ".."))
	if err != nil {
		binaryHarness.err = fmt.Errorf("リポジトリルートの実体パスを取得: %w", err)
		return
	}
	rawTempDir, err := os.MkdirTemp("", "context-cli-e2e-*")
	if err != nil {
		binaryHarness.err = fmt.Errorf("一時ディレクトリを作成: %w", err)
		return
	}
	tempDir, err := filepath.EvalSymlinks(rawTempDir)
	if err != nil {
		_ = os.RemoveAll(rawTempDir)
		binaryHarness.err = fmt.Errorf("一時ディレクトリの実体パスを取得: %w", err)
		return
	}
	binaryHarness.tempDir = tempDir
	binaryHarness.binaryPath = filepath.Join(tempDir, "context")

	ctx, cancel := context.WithTimeout(context.Background(), binaryBuildTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "build", "-o", binaryHarness.binaryPath, "./cmd/context") //nolint:gosec // 固定コマンドをテスト用一時パスへ出力する。
	command.Dir = repositoryRoot
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			binaryHarness.err = fmt.Errorf(
				"go buildがタイムアウト: %w\nstdout:\n%s\nstderr:\n%s",
				ctx.Err(),
				stdout.String(),
				stderr.String(),
			)
			return
		}
		binaryHarness.err = fmt.Errorf(
			"go buildに失敗: %w\nstdout:\n%s\nstderr:\n%s",
			err,
			stdout.String(),
			stderr.String(),
		)
	}
}

func runContextProcess(
	t *testing.T,
	request processRequest,
) processResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, contextBinary(t), request.args...) //nolint:gosec // テストが構築した実バイナリだけを起動する。
	command.Dir = request.workingDirectory
	command.Env = []string{"XDG_CONFIG_HOME=" + request.xdgConfigHome}
	command.Stdin = strings.NewReader(request.stdin)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	result := processResult{
		exitCode: 0,
		stdout:   stdout.String(),
		stderr:   stderr.String(),
	}
	if ctx.Err() != nil {
		t.Fatalf(
			"contextプロセスがタイムアウトしました\nstdout:\n%s\nstderr:\n%s",
			result.stdout,
			result.stderr,
		)
	}
	if err == nil {
		return result
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.exitCode = exitErr.ExitCode()
		return result
	}
	t.Fatalf(
		"contextプロセスを起動できません: %v\nstdout:\n%s\nstderr:\n%s",
		err,
		result.stdout,
		result.stderr,
	)
	return processResult{}
}

func runContextTerminal(
	t *testing.T,
	request processRequest,
	steps []terminalStep,
) processResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, contextBinary(t), request.args...) //nolint:gosec // テストが構築した実バイナリだけを起動する。
	command.Dir = request.workingDirectory
	command.Env = []string{
		"XDG_CONFIG_HOME=" + request.xdgConfigHome,
		"TERM=xterm-256color",
		"LANG=C.UTF-8",
	}
	terminal, err := pty.StartWithSize(command, &pty.Winsize{Rows: 30, Cols: 120})
	if err != nil {
		t.Fatalf("PTY上でcontextプロセスを起動できません: %v", err)
	}
	defer func() { _ = terminal.Close() }()

	var output bytes.Buffer
	var outputMu sync.Mutex
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		buffer := make([]byte, 4096)
		for {
			count, readErr := terminal.Read(buffer)
			if count > 0 {
				outputMu.Lock()
				_, _ = output.Write(buffer[:count])
				outputMu.Unlock()
			}
			if readErr != nil {
				return
			}
		}
	}()

	for _, step := range steps {
		waitForTerminalOutput(ctx, t, &terminalOutput{
			buffer: &output,
			mutex:  &outputMu,
		}, step.waitFor)
		if _, err := terminal.Write([]byte(step.input)); err != nil {
			t.Fatalf("PTYへ入力できません: %v", err)
		}
	}

	waitErr := command.Wait()
	_ = terminal.Close()
	<-readDone
	outputMu.Lock()
	combined := output.String()
	outputMu.Unlock()
	result := processResult{stdout: combined}
	if ctx.Err() != nil {
		t.Fatalf("contextプロセスがタイムアウトしました\noutput:\n%s", combined)
	}
	if waitErr == nil {
		return result
	}
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		result.exitCode = exitErr.ExitCode()
		return result
	}
	t.Fatalf("contextプロセスの待機に失敗しました: %v\noutput:\n%s", waitErr, combined)
	return processResult{}
}

type terminalOutput struct {
	buffer *bytes.Buffer
	mutex  *sync.Mutex
}

func waitForTerminalOutput(
	ctx context.Context,
	t *testing.T,
	output *terminalOutput,
	expected string,
) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		output.mutex.Lock()
		current := output.buffer.String()
		output.mutex.Unlock()
		if strings.Contains(current, expected) {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("プロンプト %q を待機中にタイムアウトしました\noutput:\n%s", expected, current)
		case <-ticker.C:
		}
	}
}
