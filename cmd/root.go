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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/k1LoW/runblock/parser"
	"github.com/k1LoW/runblock/runner"
	"github.com/k1LoW/runblock/version"
	"github.com/spf13/cobra"
)

var (
	defaultCommand string
	watch          bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "runblock [MARKDOWN_FILE]",
	Short: "Execute code blocks in Markdown files",
	Long: `runblock parses Markdown files and executes code blocks using specified commands.

Code blocks can specify a command in the info string after the language:

    ` + "```go /usr/bin/gofmt" + `
    package main
    ` + "```" + `

Template variables are supported:
  {{lang}}    - Language identifier of the code block
  {{content}} - Content of the code block
  {{i}}       - Index of the code block (0-based)

Environment variables are also set:
  CODEBLOCK_LANG    - Language identifier
  CODEBLOCK_CONTENT - Content of the code block
  CODEBLOCK_INDEX   - Index of the code block (0-based)

The code block content is also passed via stdin.`,
	Args:    cobra.MaximumNArgs(1),
	RunE:    run,
	Version: version.Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&defaultCommand, "default-command", "",
		"default command for code blocks without explicit command")
	rootCmd.Flags().BoolVarP(&watch, "watch", "w", false,
		"watch the file for changes and re-run on modifications")
}

func run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Watch mode requires a file argument
	if watch && len(args) == 0 {
		return errors.New("--watch requires a file argument (cannot watch stdin)")
	}

	if watch {
		return runWatch(ctx, args[0])
	}

	return runOnce(ctx, args)
}

func runOnce(ctx context.Context, args []string) error {
	// Read input
	var source []byte
	var err error

	if len(args) == 0 {
		// Read from stdin
		source, err = io.ReadAll(os.Stdin)
	} else {
		// Read from file
		source, err = os.ReadFile(args[0])
	}
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	// Parse markdown
	blocks, err := parser.Parse(source)
	if err != nil {
		return fmt.Errorf("failed to parse markdown: %w", err)
	}

	// Execute code blocks
	r := runner.New(defaultCommand)

	return r.RunAll(ctx, blocks)
}

func runWatch(ctx context.Context, filePath string) error {
	// Get the absolute path of the file
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	dir := filepath.Dir(absPath)
	fileName := filepath.Base(absPath)

	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }() //nostyle:handlerrors

	// Watch the directory (more robust for editor behavior)
	if err := watcher.Add(dir); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Run once initially
	fmt.Fprintf(os.Stderr, "Watching %s for changes...\n", absPath)
	if err := runOnce(ctx, []string{filePath}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	// Batch events like deck does
	var events []fsnotify.Event

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-sigCh:
			fmt.Fprintln(os.Stderr, "\nStopping watch...")
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			// Drain all stacked events
			events = append(events, event)
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		case <-time.After(time.Second):
			// Check if our file was modified
			fileModified := false
			for _, event := range events {
				if filepath.Base(event.Name) == fileName &&
					(event.Op&fsnotify.Write == fsnotify.Write ||
						event.Op&fsnotify.Create == fsnotify.Create) {
					fileModified = true
					break
				}
			}
			events = nil

			if !fileModified {
				continue
			}

			fmt.Fprintf(os.Stderr, "\nFile changed, re-running...\n")
			if err := runOnce(ctx, []string{filePath}); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}
	}
}
