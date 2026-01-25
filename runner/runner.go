/*
Copyright (c) 2026 Ken'ichiro Oyama <k1lowxb@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/k1LoW/runblock/parser"
)

// Runner executes commands for code blocks.
type Runner struct {
	DefaultCommand string
	Commands       map[string]string // language -> command
	Stdout         io.Writer
	Stderr         io.Writer
}

// New creates a new Runner with the given default command and language-specific commands.
func New(defaultCommand string, commands map[string]string) *Runner {
	return &Runner{
		DefaultCommand: defaultCommand,
		Commands:       commands,
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
	}
}

// Run executes the command for a code block.
// index is the 0-based index of the code block.
func (r *Runner) Run(ctx context.Context, block parser.CodeBlock, index int) error {
	// Determine command to use (priority: block command > language command > default command)
	cmd := block.Command
	if cmd == "" && r.Commands != nil {
		cmd = r.Commands[block.Language]
	}
	if cmd == "" {
		cmd = r.DefaultCommand
	}
	if cmd == "" {
		// No command specified, skip this block
		return nil
	}

	// Expand template variables
	store := map[string]any{
		"lang":    block.Language,
		"content": block.Content,
		"i":       index,
	}
	expandedCmd, err := ExpandTemplate(cmd, store)
	if err != nil {
		return fmt.Errorf("failed to expand template: %w", err)
	}

	// Skip if expanded command is empty
	expandedCmd = strings.TrimSpace(expandedCmd)
	if expandedCmd == "" {
		return nil
	}

	// Build command
	name, args, err := BuildCommand(expandedCmd)
	if err != nil {
		return fmt.Errorf("failed to build command: %w", err)
	}

	// Execute command
	execCmd := exec.CommandContext(ctx, name, args...)
	execCmd.Stdin = strings.NewReader(block.Content)
	execCmd.Stdout = r.Stdout
	execCmd.Stderr = r.Stderr

	// Set environment variables
	execCmd.Env = append(os.Environ(),
		"CODEBLOCK_LANG="+block.Language,
		"CODEBLOCK_CONTENT="+block.Content,
		fmt.Sprintf("CODEBLOCK_INDEX=%d", index),
	)

	return execCmd.Run()
}

// RunAll executes commands for all code blocks.
func (r *Runner) RunAll(ctx context.Context, blocks []parser.CodeBlock) error {
	for i, block := range blocks {
		if err := r.Run(ctx, block, i); err != nil {
			return fmt.Errorf("failed to execute code block %d: %w", i+1, err)
		}
	}
	return nil
}

// celExprReg is a regular expression to match {{expression}} patterns.
var celExprReg = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// ExpandTemplate expands template expressions in the format {{CEL expression}} with values from the store.
// It supports CEL (Common Expression Language) expressions within the template.
func ExpandTemplate(template string, store map[string]any) (string, error) {
	// Create CEL environment with store variables
	env, err := createCELEnv(store)
	if err != nil {
		return "", fmt.Errorf("failed to create CEL environment: %w", err)
	}

	var expandErr error
	result := celExprReg.ReplaceAllStringFunc(template, func(match string) string {
		// Extract CEL expression without {{ }}
		expr := strings.TrimSpace(match[2 : len(match)-2])

		// Compile and evaluate CEL expression
		ast, issues := env.Compile(expr)
		if issues != nil && issues.Err() != nil {
			expandErr = fmt.Errorf("template compilation error for '{{%s}}': %w", expr, issues.Err())
			return match // Return original match on error
		}

		prg, err := env.Program(ast)
		if err != nil {
			expandErr = fmt.Errorf("template program creation error for '{{%s}}': %w", expr, err)
			return match // Return original match on error
		}

		out, _, err := prg.Eval(store)
		if err != nil {
			expandErr = fmt.Errorf("template evaluation error for '{{%s}}': %w", expr, err)
			return match // Return original match on error
		}

		// Convert result to string
		return fmt.Sprintf("%v", out.Value())
	})

	if expandErr != nil {
		return "", expandErr
	}

	return result, nil
}

// createCELEnv creates a CEL environment with all variables from the store.
func createCELEnv(store map[string]any) (*cel.Env, error) {
	var options []cel.EnvOption

	// Add each top-level store key as a CEL variable
	for key, value := range store {
		celType := inferCELType(value)
		options = append(options, cel.Variable(key, celType))
	}

	return cel.NewEnv(options...)
}

// inferCELType infers the CEL type from a Go value.
func inferCELType(value any) *cel.Type {
	switch value.(type) {
	case string:
		return cel.StringType
	case int, int32, int64:
		return cel.IntType
	case float32, float64:
		return cel.DoubleType
	case bool:
		return cel.BoolType
	case map[string]any:
		return cel.MapType(cel.StringType, cel.AnyType)
	case map[string]string:
		return cel.MapType(cel.StringType, cel.StringType)
	case []any:
		return cel.ListType(cel.AnyType)
	case []string:
		return cel.ListType(cel.StringType)
	default:
		return cel.AnyType
	}
}

// standaloneCommandReg matches simple standalone commands without special characters.
var standaloneCommandReg = regexp.MustCompile(`^[-_.+a-zA-Z0-9]+$`)

// BuildCommand builds a command to execute.
// For simple commands (just a command name), it returns the command directly.
// For complex commands (with arguments or pipes), it wraps them in a shell.
func BuildCommand(c string) (string, []string, error) {
	// If the string looks like a standalone command, we don't need to execute it via the shell.
	if standaloneCommandReg.MatchString(c) {
		return c, nil, nil
	}

	// Wrap in shell
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/c", c}, nil
	}

	sh := detectShell()
	return sh, []string{"-c", c}, nil
}

// detectShell detects the shell to use for command execution.
func detectShell() string {
	sh := os.Getenv("SHELL")
	if sh != "" {
		return sh
	}
	// Fallback to sh
	return "/bin/sh"
}
