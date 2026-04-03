package steps

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type OpenCodeRunner struct {
	WorkDir string
	Model   string // e.g. "minimax/MiniMax-M2.5"
}

func (o *OpenCodeRunner) RunPrompt(prompt string) (string, error) {
	binary := filepath.Join(os.Getenv("HOME"), ".opencode", "bin", "opencode")
	model := o.Model
	if model == "" {
		model = "minimax/MiniMax-M2.5"
	}

	cmd := exec.Command(binary, "run", prompt, "-m", model, "--dir", o.WorkDir)
	cmd.Dir = o.WorkDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("exit code %d: %v", cmd.ProcessState.ExitCode(), err)
	}

	return string(output), nil
}

func (o *OpenCodeRunner) ReadHTMLFiles(dir string) (string, error) {
	var result string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".html" {
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			result += fmt.Sprintf("=== %s ===\n%s\n\n", path, string(content))
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	return result, nil
}
