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
package parser

import (
	"testing"
)

func TestParseInfoString(t *testing.T) {
	tests := []struct {
		name        string
		info        string
		wantLang    string
		wantCommand string
	}{
		{
			name:        "language only",
			info:        "go",
			wantLang:    "go",
			wantCommand: "",
		},
		{
			name:        "language with command",
			info:        "go /usr/bin/gofmt",
			wantLang:    "go",
			wantCommand: "/usr/bin/gofmt",
		},
		{
			name:        "language with command and template",
			info:        "go /path/to/cmd {{lang}} {{content}}",
			wantLang:    "go",
			wantCommand: "/path/to/cmd {{lang}} {{content}}",
		},
		{
			name:        "empty info string",
			info:        "",
			wantLang:    "",
			wantCommand: "",
		},
		{
			name:        "whitespace only",
			info:        "   ",
			wantLang:    "",
			wantCommand: "",
		},
		{
			name:        "language with extra spaces",
			info:        "python   python3 {{content}}",
			wantLang:    "python",
			wantCommand: "python3 {{content}}",
		},
		{
			name:        "language with leading/trailing spaces",
			info:        "  sh  echo hello  ",
			wantLang:    "sh",
			wantCommand: "echo hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLang, gotCommand := ParseInfoString(tt.info)
			if gotLang != tt.wantLang {
				t.Errorf("ParseInfoString() gotLang = %q, want %q", gotLang, tt.wantLang) //nostyle:errorstrings
			}
			if gotCommand != tt.wantCommand {
				t.Errorf("ParseInfoString() gotCommand = %q, want %q", gotCommand, tt.wantCommand) //nostyle:errorstrings
			}
		})
	}
}

func TestParse_BasicCodeBlock(t *testing.T) {
	source := []byte("# Test\n\n```go\npackage main\n```\n")

	blocks, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("Parse() got %d blocks, want 1", len(blocks))
	}

	if blocks[0].Language != "go" {
		t.Errorf("blocks[0].Language = %q, want %q", blocks[0].Language, "go")
	}
	if blocks[0].Command != "" {
		t.Errorf("blocks[0].Command = %q, want empty", blocks[0].Command)
	}
	if blocks[0].Content != "package main\n" {
		t.Errorf("blocks[0].Content = %q, want %q", blocks[0].Content, "package main\n")
	}
}

func TestParse_CodeBlockWithCommand(t *testing.T) {
	source := []byte("```go /usr/bin/gofmt\npackage main\n\nfunc main() {}\n```\n")

	blocks, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("Parse() got %d blocks, want 1", len(blocks))
	}

	if blocks[0].Language != "go" {
		t.Errorf("blocks[0].Language = %q, want %q", blocks[0].Language, "go")
	}
	if blocks[0].Command != "/usr/bin/gofmt" {
		t.Errorf("blocks[0].Command = %q, want %q", blocks[0].Command, "/usr/bin/gofmt")
	}
	want := "package main\n\nfunc main() {}\n"
	if blocks[0].Content != want {
		t.Errorf("blocks[0].Content = %q, want %q", blocks[0].Content, want)
	}
}

func TestParse_MultipleCodeBlocks(t *testing.T) {
	source := []byte(`# Test

` + "```go /usr/bin/gofmt" + `
package main
` + "```" + `

Some text

` + "```python python3 {{content}}" + `
print("hello")
` + "```" + `

` + "```sh" + `
echo "hello"
` + "```" + `
`)

	blocks, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(blocks) != 3 {
		t.Fatalf("Parse() got %d blocks, want 3", len(blocks))
	}

	// First block
	if blocks[0].Language != "go" {
		t.Errorf("blocks[0].Language = %q, want %q", blocks[0].Language, "go")
	}
	if blocks[0].Command != "/usr/bin/gofmt" {
		t.Errorf("blocks[0].Command = %q, want %q", blocks[0].Command, "/usr/bin/gofmt")
	}

	// Second block
	if blocks[1].Language != "python" {
		t.Errorf("blocks[1].Language = %q, want %q", blocks[1].Language, "python")
	}
	if blocks[1].Command != "python3 {{content}}" {
		t.Errorf("blocks[1].Command = %q, want %q", blocks[1].Command, "python3 {{content}}")
	}

	// Third block
	if blocks[2].Language != "sh" {
		t.Errorf("blocks[2].Language = %q, want %q", blocks[2].Language, "sh")
	}
	if blocks[2].Command != "" {
		t.Errorf("blocks[2].Command = %q, want empty", blocks[2].Command)
	}
}

func TestParse_EmptyInfoString(t *testing.T) {
	source := []byte("```\nsome content\n```\n")

	blocks, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("Parse() got %d blocks, want 1", len(blocks))
	}

	if blocks[0].Language != "" {
		t.Errorf("blocks[0].Language = %q, want empty", blocks[0].Language)
	}
	if blocks[0].Command != "" {
		t.Errorf("blocks[0].Command = %q, want empty", blocks[0].Command)
	}
	if blocks[0].Content != "some content\n" {
		t.Errorf("blocks[0].Content = %q, want %q", blocks[0].Content, "some content\n")
	}
}

func TestParse_NoCodeBlocks(t *testing.T) {
	source := []byte("# Title\n\nSome paragraph text.\n\n- item 1\n- item 2\n")

	blocks, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(blocks) != 0 {
		t.Fatalf("Parse() got %d blocks, want 0", len(blocks))
	}
}

func TestParse_CodeBlockWithTemplateVariables(t *testing.T) {
	source := []byte("```go /path/to/cmd {{lang}} {{content}}\npackage main\n```\n")

	blocks, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("Parse() got %d blocks, want 1", len(blocks))
	}

	if blocks[0].Command != "/path/to/cmd {{lang}} {{content}}" {
		t.Errorf("blocks[0].Command = %q, want %q", blocks[0].Command, "/path/to/cmd {{lang}} {{content}}")
	}
}
