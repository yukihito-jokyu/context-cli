package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/yukihito-jokyu/context-cli/internal/distribution"
	"github.com/yukihito-jokyu/context-cli/internal/skillcatalog"
)

var errPromptInteractionTest = errors.New("interaction failed")

func TestHuhPromptPreservesUserAbort(t *testing.T) {
	input := bytes.NewBufferString("input")
	output := &bytes.Buffer{}
	prompt := &huhPrompt{
		input:  input,
		output: output,
		run: func(*huh.Form) error {
			return huh.ErrUserAborted
		},
	}

	_, err := prompt.SelectProject([]skillcatalog.Candidate{{Name: "project"}}, "")
	if !errors.Is(err, huh.ErrUserAborted) {
		t.Fatalf("SelectProject() error = %v, want huh.ErrUserAborted", err)
	}
}

func TestHuhPromptWrapsInteractionError(t *testing.T) {
	prompt := &huhPrompt{
		input:  &bytes.Buffer{},
		output: &bytes.Buffer{},
		run: func(*huh.Form) error {
			return errPromptInteractionTest
		},
	}

	_, err := prompt.SelectSkills(SkillKindProject, []skillcatalog.Candidate{{Name: "skill"}}, nil)
	if !errors.Is(err, ErrPrompt) || !errors.Is(err, errPromptInteractionTest) {
		t.Fatalf("SelectSkills() error = %v, want ErrPrompt wrapping cause", err)
	}
}

func TestCommonSkillsConfirmationDefaultsToRejection(t *testing.T) {
	selected := false
	field := newCommonSkillsConfirm(&selected)
	if got := field.GetValue(); got != false {
		t.Fatalf("GetValue() = %v, want false", got)
	}
}

func TestDestinationValidationRequiresSelection(t *testing.T) {
	if err := validateDestinations(nil); !errors.Is(err, ErrDestinationRequired) {
		t.Fatalf("validateDestinations(nil) error = %v, want ErrDestinationRequired", err)
	}
	if err := validateDestinations([]distribution.Destination{distribution.DestinationCodex}); err != nil {
		t.Fatalf("validateDestinations() error = %v", err)
	}
}
