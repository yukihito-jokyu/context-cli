package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yukihito-jokyu/context-cli/internal/repository"
)

var (
	errValidationTest = errors.New("validation failed")
	errConfigLoadTest = errors.New("config load failed")
	errConfigSaveTest = errors.New("config save failed")
	errInputTest      = errors.New("input failed")
)

type initRunCase struct {
	name              string
	validator         *stubRepositoryValidator
	currentRepository string
	configErr         error
	setErr            error
	input             string
	inputErr          error
	outputErr         error
	outputFailCall    int
	wantErr           error
	wantConfigCalls   int
	wantSetCalls      int
	wantSaved         string
	wantOutput        string
	wantReadCalls     int
	wantErrorText     string
}

func TestInitOptionsRun(t *testing.T) {
	currentPath := "/current/context"
	validatedPath := "/validated/context"
	confirmation := "Current context repository: /current/context\n" +
		"New context repository: /validated/context\n" +
		"変更しますか? [y/N] "
	success := "Successfully initialized context repository at: /validated/context\n"
	confirmationWriteErr := &os.PathError{
		Op:   "write",
		Path: "/secret/confirmation/output",
		Err:  fs.ErrPermission,
	}
	successWriteErr := &os.PathError{
		Op:   "write",
		Path: "/secret/success/output",
		Err:  fs.ErrPermission,
	}

	tests := []initRunCase{
		{
			name:            "初回設定は確認せず保存して表示する",
			validator:       &stubRepositoryValidator{validatedPath: validatedPath},
			wantConfigCalls: 1,
			wantSetCalls:    1,
			wantSaved:       validatedPath,
			wantOutput:      success,
		},
		{
			name:              "同一パスは確認と保存を省略して表示する",
			validator:         &stubRepositoryValidator{validatedPath: validatedPath},
			currentRepository: validatedPath,
			wantConfigCalls:   1,
			wantSaved:         validatedPath,
			wantOutput:        success,
		},
		{
			name:              "異なるパスは小文字yで変更する",
			validator:         &stubRepositoryValidator{validatedPath: validatedPath},
			currentRepository: currentPath,
			input:             "y\n",
			wantConfigCalls:   1,
			wantSetCalls:      1,
			wantSaved:         validatedPath,
			wantOutput:        confirmation + success,
			wantReadCalls:     1,
		},
		{
			name:              "前後の空白を除いた小文字yで変更する",
			validator:         &stubRepositoryValidator{validatedPath: validatedPath},
			currentRepository: currentPath,
			input:             "  y \t\n",
			wantConfigCalls:   1,
			wantSetCalls:      1,
			wantSaved:         validatedPath,
			wantOutput:        confirmation + success,
			wantReadCalls:     1,
		},
		{
			name:              "大文字Yでは変更をキャンセルする",
			validator:         &stubRepositoryValidator{validatedPath: validatedPath},
			currentRepository: currentPath,
			input:             "Y\n",
			wantErr:           ErrRepositoryChangeCanceled,
			wantConfigCalls:   1,
			wantSaved:         currentPath,
			wantOutput:        confirmation,
			wantReadCalls:     1,
		},
		{
			name:              "yesでは変更をキャンセルする",
			validator:         &stubRepositoryValidator{validatedPath: validatedPath},
			currentRepository: currentPath,
			input:             "yes\n",
			wantErr:           ErrRepositoryChangeCanceled,
			wantConfigCalls:   1,
			wantSaved:         currentPath,
			wantOutput:        confirmation,
			wantReadCalls:     1,
		},
		{
			name:              "空入力では変更をキャンセルする",
			validator:         &stubRepositoryValidator{validatedPath: validatedPath},
			currentRepository: currentPath,
			input:             "\n",
			wantErr:           ErrRepositoryChangeCanceled,
			wantConfigCalls:   1,
			wantSaved:         currentPath,
			wantOutput:        confirmation,
			wantReadCalls:     1,
		},
		{
			name:              "任意入力では変更をキャンセルする",
			validator:         &stubRepositoryValidator{validatedPath: validatedPath},
			currentRepository: currentPath,
			input:             "n\n",
			wantErr:           ErrRepositoryChangeCanceled,
			wantConfigCalls:   1,
			wantSaved:         currentPath,
			wantOutput:        confirmation,
			wantReadCalls:     1,
		},
		{
			name:              "空のEOFでは変更をキャンセルする",
			validator:         &stubRepositoryValidator{validatedPath: validatedPath},
			currentRepository: currentPath,
			wantErr:           ErrRepositoryChangeCanceled,
			wantConfigCalls:   1,
			wantSaved:         currentPath,
			wantOutput:        confirmation,
			wantReadCalls:     1,
		},
		{
			name:              "改行なしのyとEOFでは変更をキャンセルする",
			validator:         &stubRepositoryValidator{validatedPath: validatedPath},
			currentRepository: currentPath,
			input:             "y",
			wantErr:           ErrRepositoryChangeCanceled,
			wantConfigCalls:   1,
			wantSaved:         currentPath,
			wantOutput:        confirmation,
			wantReadCalls:     2,
		},
		{
			name:              "入力エラーでは変更をキャンセルする",
			validator:         &stubRepositoryValidator{validatedPath: validatedPath},
			currentRepository: currentPath,
			inputErr:          errInputTest,
			wantErr:           ErrRepositoryChangeCanceled,
			wantConfigCalls:   1,
			wantSaved:         currentPath,
			wantOutput:        confirmation,
			wantReadCalls:     1,
		},
		{
			name:              "確認出力失敗では入力と保存を行わない",
			validator:         &stubRepositoryValidator{validatedPath: validatedPath},
			currentRepository: currentPath,
			input:             "y\n",
			outputErr:         confirmationWriteErr,
			outputFailCall:    1,
			wantErr:           confirmationWriteErr,
			wantConfigCalls:   1,
			wantSaved:         currentPath,
			wantErrorText:     "failed to write repository change confirmation",
		},
		{
			name:              "承認後の保存失敗では成功を表示しない",
			validator:         &stubRepositoryValidator{validatedPath: validatedPath},
			currentRepository: currentPath,
			input:             "y\n",
			setErr:            errConfigSaveTest,
			wantErr:           errConfigSaveTest,
			wantConfigCalls:   1,
			wantSetCalls:      1,
			wantSaved:         currentPath,
			wantOutput:        confirmation,
			wantReadCalls:     1,
		},
		{
			name:            "初回設定の成功出力失敗は保存を戻さない",
			validator:       &stubRepositoryValidator{validatedPath: validatedPath},
			outputErr:       successWriteErr,
			outputFailCall:  1,
			wantErr:         successWriteErr,
			wantConfigCalls: 1,
			wantSetCalls:    1,
			wantSaved:       validatedPath,
			wantErrorText:   "failed to write initialization success message",
		},
		{
			name:              "同一パスの成功出力失敗を返す",
			validator:         &stubRepositoryValidator{validatedPath: validatedPath},
			currentRepository: validatedPath,
			outputErr:         successWriteErr,
			outputFailCall:    1,
			wantErr:           successWriteErr,
			wantConfigCalls:   1,
			wantSaved:         validatedPath,
			wantErrorText:     "failed to write initialization success message",
		},
		{
			name:              "変更後の成功出力失敗は保存を戻さない",
			validator:         &stubRepositoryValidator{validatedPath: validatedPath},
			currentRepository: currentPath,
			input:             "y\n",
			outputErr:         successWriteErr,
			outputFailCall:    2,
			wantErr:           successWriteErr,
			wantConfigCalls:   1,
			wantSetCalls:      1,
			wantSaved:         validatedPath,
			wantOutput:        confirmation,
			wantReadCalls:     1,
			wantErrorText:     "failed to write initialization success message",
		},
		{
			name:      "検証失敗時は設定を取得しない",
			validator: &stubRepositoryValidator{err: errValidationTest},
			wantErr:   errValidationTest,
		},
		{
			name:            "設定取得失敗を返す",
			validator:       &stubRepositoryValidator{validatedPath: validatedPath},
			configErr:       errConfigLoadTest,
			wantErr:         errConfigLoadTest,
			wantConfigCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runInitCase(t, tt)
		})
	}
}

func runInitCase(t *testing.T, tt initRunCase) {
	t.Helper()
	output := &recordingWriter{err: tt.outputErr, failCall: tt.outputFailCall}
	input := &recordingReader{reader: strings.NewReader(tt.input), err: tt.inputErr}
	config := &recordingConfig{savedPath: tt.currentRepository, setErr: tt.setErr}
	configCalls := 0
	factory := &Factory{
		IOIn:                input,
		IOOut:               output,
		RepositoryValidator: tt.validator,
		Config: func() (Config, error) {
			configCalls++
			return config, tt.configErr
		},
	}
	options := &InitOptions{Factory: factory, RepoPath: "input/context"}

	err := options.Run()
	if !errors.Is(err, tt.wantErr) {
		t.Fatalf("Run() error = %v, want errors.Is(_, %v)", err, tt.wantErr)
	}
	if configCalls != tt.wantConfigCalls {
		t.Errorf("Config calls = %d, want %d", configCalls, tt.wantConfigCalls)
	}
	if config.setCalls != tt.wantSetCalls {
		t.Errorf("SetContextRepository calls = %d, want %d", config.setCalls, tt.wantSetCalls)
	}
	if config.savedPath != tt.wantSaved {
		t.Errorf("saved path = %q, want %q", config.savedPath, tt.wantSaved)
	}
	if output.String() != tt.wantOutput {
		t.Errorf("output = %q, want %q", output.String(), tt.wantOutput)
	}
	if input.readCalls != tt.wantReadCalls {
		t.Errorf("input read calls = %d, want %d", input.readCalls, tt.wantReadCalls)
	}
	if tt.wantErrorText != "" && (err == nil || err.Error() != tt.wantErrorText) {
		t.Errorf("error text = %q, want %q", err, tt.wantErrorText)
	}
	if tt.outputErr != nil && err != nil && strings.Contains(err.Error(), tt.outputErr.Error()) {
		t.Errorf("error message leaks internal I/O error: %q", err)
	}
	if tt.validator.input != "input/context" {
		t.Errorf("validator input = %q, want %q", tt.validator.input, "input/context")
	}
}

func TestInitOptionsRunDoesNotLeakValidationPaths(t *testing.T) {
	inputPath := filepath.Join("..", "parent", "context")
	absolutePath := filepath.Join(string(filepath.Separator), "secret", "parent", "context")
	internalPath := filepath.Join(string(filepath.Separator), "internal", "failure", "path")
	validationErr := &repository.ValidationError{
		Kind:   repository.ErrIO,
		Target: "repository parent",
		Err:    &os.PathError{Op: "lstat", Path: internalPath, Err: fs.ErrPermission},
	}
	configCalls := 0
	factory := &Factory{
		IOOut:               new(bytes.Buffer),
		RepositoryValidator: &stubRepositoryValidator{err: validationErr},
		Config: func() (Config, error) {
			configCalls++
			return &recordingConfig{}, nil
		},
	}

	err := (&InitOptions{Factory: factory, RepoPath: inputPath}).Run()
	if !errors.Is(err, repository.ErrIO) {
		t.Fatalf("Run() error = %v, want ErrIO", err)
	}
	if configCalls != 0 {
		t.Fatalf("Config calls = %d, want 0", configCalls)
	}
	for _, secret := range []string{inputPath, absolutePath, internalPath} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("error message leaks path %q: %q", secret, err)
		}
	}
}

type stubRepositoryValidator struct {
	validatedPath string
	err           error
	input         string
}

func (v *stubRepositoryValidator) Validate(path string) (string, error) {
	v.input = path
	return v.validatedPath, v.err
}

type recordingConfig struct {
	savedPath string
	setErr    error
	setCalls  int
}

func (c *recordingConfig) GetContextRepository() string {
	return c.savedPath
}

func (c *recordingConfig) SetContextRepository(path string) error {
	c.setCalls++
	if c.setErr != nil {
		return c.setErr
	}
	c.savedPath = path
	return nil
}

type recordingReader struct {
	reader    io.Reader
	err       error
	readCalls int
}

func (r *recordingReader) Read(p []byte) (int, error) {
	r.readCalls++
	if r.err != nil {
		return 0, r.err
	}
	n, err := r.reader.Read(p)
	if err != nil {
		return n, fmt.Errorf("テスト入力の読み取りに失敗しました: %w", err)
	}
	return n, nil
}

type recordingWriter struct {
	buffer   bytes.Buffer
	err      error
	failCall int
	calls    int
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.err != nil && w.calls == w.failCall {
		return 0, w.err
	}
	n, _ := w.buffer.Write(p)
	return n, nil
}

func (w *recordingWriter) String() string {
	return w.buffer.String()
}
