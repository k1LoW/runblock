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
package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/k1LoW/runblock/parser"
	"github.com/k1LoW/runblock/runner"
)

func TestRunBlock_FromFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	testFile := filepath.Join("..", "testdata", "basic.md")
	source, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	blocks, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("failed to parse markdown: %v", err)
	}

	var stdout, stderr bytes.Buffer
	r := &runner.Runner{
		DefaultCommand: "",
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	for i, block := range blocks {
		if err := r.Run(t.Context(), block, i); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}

	got := stdout.String()
	if !strings.Contains(got, "hello world") {
		t.Errorf("stdout does not contain 'hello world': %q", got)
	}
	if !strings.Contains(got, "second block") {
		t.Errorf("stdout does not contain 'second block': %q", got)
	}
}

func TestRunBlock_FromStdin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	// Simulate stdin content
	content := "```sh cat\nstdin content\n```\n"
	source := []byte(content)

	blocks, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("failed to parse markdown: %v", err)
	}

	var stdout, stderr bytes.Buffer
	r := &runner.Runner{
		DefaultCommand: "",
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	for i, block := range blocks {
		if err := r.Run(t.Context(), block, i); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}

	want := "stdin content\n"
	if got := stdout.String(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRunBlock_WithDefaultCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	testFile := filepath.Join("..", "testdata", "mixed.md")
	source, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	blocks, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("failed to parse markdown: %v", err)
	}

	var stdout, stderr bytes.Buffer
	r := &runner.Runner{
		DefaultCommand: "cat", // Default command for blocks without command
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	for i, block := range blocks {
		if err := r.Run(t.Context(), block, i); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}

	got := stdout.String()
	// Block with command
	if !strings.Contains(got, "block with command") {
		t.Errorf("stdout does not contain 'block with command': %q", got)
	}
	// Block without command (should use default)
	if !strings.Contains(got, "block without command") {
		t.Errorf("stdout does not contain 'block without command': %q", got)
	}
	// Another block with command
	if !strings.Contains(got, "another block") {
		t.Errorf("stdout does not contain 'another block': %q", got)
	}
}

func TestRunBlock_MixedBlocks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	testFile := filepath.Join("..", "testdata", "with_template.md")
	source, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	blocks, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("failed to parse markdown: %v", err)
	}

	var stdout, stderr bytes.Buffer
	r := &runner.Runner{
		DefaultCommand: "",
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	for i, block := range blocks {
		if err := r.Run(t.Context(), block, i); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}

	got := stdout.String()
	// First block should output "go"
	if !strings.Contains(got, "go") {
		t.Errorf("stdout does not contain 'go': %q", got)
	}
	// Second block should output "python"
	if !strings.Contains(got, "python") {
		t.Errorf("stdout does not contain 'python': %q", got)
	}
}

func TestRunBlock_CELExpression(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	// Test CEL ternary expression
	content := "```go echo {{ lang == \"\" ? \"none\" : lang }}\npackage main\n```\n"
	source := []byte(content)

	blocks, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("failed to parse markdown: %v", err)
	}

	var stdout, stderr bytes.Buffer
	r := &runner.Runner{
		DefaultCommand: "",
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	for i, block := range blocks {
		if err := r.Run(t.Context(), block, i); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}

	got := strings.TrimSpace(stdout.String())
	if got != "go" {
		t.Errorf("stdout = %q, want %q", got, "go")
	}
}

func TestRunBlock_CELExpressionEmptyLang(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	// Test CEL ternary expression with empty lang using default command
	// Note: When lang is empty, we can't specify command in info string
	// So we use default command with CEL expression
	content := "```\nsome content\n```\n"
	source := []byte(content)

	blocks, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("failed to parse markdown: %v", err)
	}

	var stdout, stderr bytes.Buffer
	r := &runner.Runner{
		DefaultCommand: `echo {{ lang == "" ? "none" : lang }}`,
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	for i, block := range blocks {
		if err := r.Run(t.Context(), block, i); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}

	got := strings.TrimSpace(stdout.String())
	if got != "none" {
		t.Errorf("stdout = %q, want %q", got, "none")
	}
}

func TestRunOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	testFile := filepath.Join("..", "testdata", "basic.md")

	// Capture original stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe() //nostyle:handlerrors
	os.Stdout = w

	// Reset defaultCommand
	defaultCommand = ""

	err := runOnce(t.Context(), []string{testFile})
	if err != nil {
		t.Fatalf("runOnce() error = %v", err)
	}

	_ = w.Close() //nostyle:handlerrors
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r) //nostyle:handlerrors
	got := buf.String()

	if !strings.Contains(got, "hello world") {
		t.Errorf("stdout does not contain 'hello world': %q", got)
	}
}
