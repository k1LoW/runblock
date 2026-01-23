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
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// CodeBlock represents a fenced code block extracted from Markdown.
type CodeBlock struct {
	Language string // Language identifier (e.g., "go", "python")
	Command  string // Command to execute (e.g., "/path/to/cmd {{lang}} {{content}}")
	Content  string // Content of the code block
}

// Parse parses Markdown source and extracts fenced code blocks.
func Parse(source []byte) ([]CodeBlock, error) { //nostyle:repetition
	md := goldmark.New()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	var blocks []CodeBlock

	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		fcb, ok := n.(*ast.FencedCodeBlock)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Extract info string and parse language/command
		var info string
		if fcb.Info != nil {
			info = string(fcb.Info.Segment.Value(source))
		}

		lang, cmd := ParseInfoString(info)

		// Extract content from lines
		var content strings.Builder
		lines := fcb.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			content.Write(line.Value(source))
		}

		blocks = append(blocks, CodeBlock{
			Language: lang,
			Command:  cmd,
			Content:  content.String(),
		})

		return ast.WalkContinue, nil
	})

	if err != nil {
		return nil, err
	}

	return blocks, nil
}

// ParseInfoString parses the info string of a fenced code block.
// It returns the language identifier and the command (if any).
// Format: "language [command]"
// Example: "go /usr/bin/gofmt {{content}}" -> ("go", "/usr/bin/gofmt {{content}}")
func ParseInfoString(info string) (language, command string) { //nostyle:repetition
	info = strings.TrimSpace(info)
	if info == "" {
		return "", ""
	}

	// Split on first space to separate language from command
	idx := strings.Index(info, " ")
	if idx < 0 {
		// No space, only language
		return info, ""
	}

	language = info[:idx]
	command = strings.TrimSpace(info[idx+1:])

	return language, command
}
