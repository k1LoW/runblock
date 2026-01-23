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
	"bytes"
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/k1LoW/runblock/parser"
)

func TestExpandTemplate_Simple(t *testing.T) {
	tests := []struct {
		name     string
		template string
		store    map[string]any
		want     string
		wantErr  bool
	}{
		{
			name:     "simple lang variable",
			template: "echo {{lang}}",
			store:    map[string]any{"lang": "go", "content": "hello"},
			want:     "echo go",
			wantErr:  false,
		},
		{
			name:     "simple content variable",
			template: "cat {{content}}",
			store:    map[string]any{"lang": "go", "content": "hello world"},
			want:     "cat hello world",
			wantErr:  false,
		},
		{
			name:     "multiple variables",
			template: "process {{lang}} {{content}}",
			store:    map[string]any{"lang": "python", "content": "print('hi')"},
			want:     "process python print('hi')",
			wantErr:  false,
		},
		{
			name:     "no variables",
			template: "echo hello",
			store:    map[string]any{"lang": "go", "content": "hello"},
			want:     "echo hello",
			wantErr:  false,
		},
		{
			name:     "index variable",
			template: "echo {{i}}",
			store:    map[string]any{"lang": "go", "content": "hello", "i": 0},
			want:     "echo 0",
			wantErr:  false,
		},
		{
			name:     "index variable non-zero",
			template: "output_{{i}}.txt",
			store:    map[string]any{"lang": "go", "content": "hello", "i": 5},
			want:     "output_5.txt",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandTemplate(tt.template, tt.store)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandTemplate() error = %v, wantErr %v", err, tt.wantErr) //nostyle:errorstrings
				return
			}
			if got != tt.want {
				t.Errorf("ExpandTemplate() = %q, want %q", got, tt.want) //nostyle:errorstrings
			}
		})
	}
}

func TestExpandTemplate_CEL(t *testing.T) {
	tests := []struct {
		name     string
		template string
		store    map[string]any
		want     string
		wantErr  bool
	}{
		{
			name:     "ternary operator - true case",
			template: `echo {{ lang == "" ? "txt" : lang }}`,
			store:    map[string]any{"lang": "go", "content": "hello"},
			want:     "echo go",
			wantErr:  false,
		},
		{
			name:     "ternary operator - false case",
			template: `echo {{ lang == "" ? "txt" : lang }}`,
			store:    map[string]any{"lang": "", "content": "hello"},
			want:     "echo txt",
			wantErr:  false,
		},
		{
			name:     "string concatenation",
			template: `echo {{ "prefix_" + lang }}`,
			store:    map[string]any{"lang": "go", "content": "hello"},
			want:     "echo prefix_go",
			wantErr:  false,
		},
		{
			name:     "string contains",
			template: `{{ content.contains("main") ? "has main" : "no main" }}`,
			store:    map[string]any{"lang": "go", "content": "func main() {}"},
			want:     "has main",
			wantErr:  false,
		},
		{
			name:     "string size",
			template: `size={{ content.size() }}`,
			store:    map[string]any{"lang": "go", "content": "hello"},
			want:     "size=5",
			wantErr:  false,
		},
		{
			name:     "index arithmetic",
			template: `block_{{ i + 1 }}`,
			store:    map[string]any{"lang": "go", "content": "hello", "i": 2},
			want:     "block_3",
			wantErr:  false,
		},
		{
			name:     "index with conditional",
			template: `{{ i == 0 ? "first" : "other" }}`,
			store:    map[string]any{"lang": "go", "content": "hello", "i": 0},
			want:     "first",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandTemplate(tt.template, tt.store)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandTemplate() error = %v, wantErr %v", err, tt.wantErr) //nostyle:errorstrings
				return
			}
			if got != tt.want {
				t.Errorf("ExpandTemplate() = %q, want %q", got, tt.want) //nostyle:errorstrings
			}
		})
	}
}

func TestBuildCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		wantName string
		wantArgs []string
		wantErr  bool
	}{
		{
			name:     "simple command",
			cmd:      "gofmt",
			wantName: "gofmt",
			wantArgs: nil,
			wantErr:  false,
		},
		{
			name:     "command with path separators",
			cmd:      "echo hello",
			wantName: func() string {
				if runtime.GOOS == "windows" {
					return "cmd"
				}
				sh := os.Getenv("SHELL")
				if sh != "" {
					return sh
				}
				return "/bin/sh"
			}(),
			wantArgs: func() []string {
				if runtime.GOOS == "windows" {
					return []string{"/c", "echo hello"}
				}
				return []string{"-c", "echo hello"}
			}(),
			wantErr: false,
		},
		{
			name:     "command with pipe",
			cmd:      "cat | grep test",
			wantName: func() string {
				if runtime.GOOS == "windows" {
					return "cmd"
				}
				sh := os.Getenv("SHELL")
				if sh != "" {
					return sh
				}
				return "/bin/sh"
			}(),
			wantArgs: func() []string {
				if runtime.GOOS == "windows" {
					return []string{"/c", "cat | grep test"}
				}
				return []string{"-c", "cat | grep test"}
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotArgs, err := BuildCommand(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildCommand() error = %v, wantErr %v", err, tt.wantErr) //nostyle:errorstrings
				return
			}
			if gotName != tt.wantName {
				t.Errorf("BuildCommand() name = %q, want %q", gotName, tt.wantName) //nostyle:errorstrings
			}
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("BuildCommand() args len = %d, want %d", len(gotArgs), len(tt.wantArgs)) //nostyle:errorstrings
				return
			}
			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("BuildCommand() args[%d] = %q, want %q", i, arg, tt.wantArgs[i]) //nostyle:errorstrings
				}
			}
		})
	}
}

func TestRun_BasicExecution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	var stdout, stderr bytes.Buffer
	r := &Runner{
		DefaultCommand: "",
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	block := parser.CodeBlock{
		Language: "sh",
		Command:  "cat",
		Content:  "hello world",
	}

	err := r.Run(context.Background(), block, 0)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := stdout.String(); got != "hello world" {
		t.Errorf("stdout = %q, want %q", got, "hello world")
	}
}

func TestRun_WithTemplateVariables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	var stdout, stderr bytes.Buffer
	r := &Runner{
		DefaultCommand: "",
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	block := parser.CodeBlock{
		Language: "go",
		Command:  "echo {{lang}}",
		Content:  "package main",
	}

	err := r.Run(context.Background(), block, 0)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got := strings.TrimSpace(stdout.String())
	if got != "go" {
		t.Errorf("stdout = %q, want %q", got, "go")
	}
}

func TestRun_WithIndex(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	var stdout, stderr bytes.Buffer
	r := &Runner{
		DefaultCommand: "",
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	block := parser.CodeBlock{
		Language: "go",
		Command:  "echo {{i}}",
		Content:  "package main",
	}

	// Test with index 3
	err := r.Run(context.Background(), block, 3)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got := strings.TrimSpace(stdout.String())
	if got != "3" {
		t.Errorf("stdout = %q, want %q", got, "3")
	}
}

func TestRun_WithIndexEnvVar(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	var stdout, stderr bytes.Buffer
	r := &Runner{
		DefaultCommand: "",
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	block := parser.CodeBlock{
		Language: "go",
		Command:  "sh -c 'echo $CODEBLOCK_INDEX'",
		Content:  "package main",
	}

	err := r.Run(context.Background(), block, 5)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got := strings.TrimSpace(stdout.String())
	if got != "5" {
		t.Errorf("stdout = %q, want %q", got, "5")
	}
}

func TestRun_StdinContent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	var stdout, stderr bytes.Buffer
	r := &Runner{
		DefaultCommand: "",
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	block := parser.CodeBlock{
		Language: "text",
		Command:  "cat",
		Content:  "line1\nline2\nline3",
	}

	err := r.Run(context.Background(), block, 0)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := "line1\nline2\nline3"
	if got := stdout.String(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRun_EnvironmentVariables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	var stdout, stderr bytes.Buffer
	r := &Runner{
		DefaultCommand: "",
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	block := parser.CodeBlock{
		Language: "go",
		Command:  "sh -c 'echo $CODEBLOCK_LANG'",
		Content:  "package main",
	}

	err := r.Run(context.Background(), block, 0)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got := strings.TrimSpace(stdout.String())
	if got != "go" {
		t.Errorf("stdout = %q, want %q", got, "go")
	}
}

func TestRun_DefaultCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	var stdout, stderr bytes.Buffer
	r := &Runner{
		DefaultCommand: "cat",
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	block := parser.CodeBlock{
		Language: "text",
		Command:  "", // No command specified
		Content:  "default command test",
	}

	err := r.Run(context.Background(), block, 0)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := "default command test"
	if got := stdout.String(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRun_NoCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	r := &Runner{
		DefaultCommand: "", // No default command
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	block := parser.CodeBlock{
		Language: "text",
		Command:  "", // No command specified
		Content:  "some content",
	}

	err := r.Run(context.Background(), block, 0)
	if err != nil {
		t.Fatalf("Run() should not return error when no command is specified, got %v", err)
	}

	// Should produce no output since no command was executed
	if got := stdout.String(); got != "" {
		t.Errorf("stdout = %q, want empty", got)
	}
}

func TestRun_SkipOnEmptyExpandedCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	r := &Runner{
		DefaultCommand: `{{ lang == "go" ? "cat" : "" }}`,
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	// This block should be skipped because lang is "python", resulting in empty command
	block := parser.CodeBlock{
		Language: "python",
		Command:  "",
		Content:  "should not appear",
	}

	err := r.Run(context.Background(), block, 0)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should produce no output since command expanded to empty string
	if got := stdout.String(); got != "" {
		t.Errorf("stdout = %q, want empty", got)
	}
}

func TestRun_ExecuteOnNonEmptyExpandedCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	var stdout, stderr bytes.Buffer
	r := &Runner{
		DefaultCommand: `{{ lang == "go" ? "cat" : "" }}`,
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	// This block should execute because lang is "go", resulting in "cat" command
	block := parser.CodeBlock{
		Language: "go",
		Command:  "",
		Content:  "should appear",
	}

	err := r.Run(context.Background(), block, 0)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := "should appear"
	if got := stdout.String(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRunAll(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows")
	}

	var stdout, stderr bytes.Buffer
	r := &Runner{
		DefaultCommand: "cat",
		Stdout:         &stdout,
		Stderr:         &stderr,
	}

	blocks := []parser.CodeBlock{
		{
			Language: "text",
			Command:  "",
			Content:  "block1",
		},
		{
			Language: "text",
			Command:  "",
			Content:  "block2",
		},
	}

	err := r.RunAll(context.Background(), blocks)
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}

	want := "block1block2"
	if got := stdout.String(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}
