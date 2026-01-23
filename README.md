# runblock

`runblock` is a tool for executing code blocks in Markdown files using external commands.

## Usage

### Basic usage

```console
$ runblock example.md
```

### Read from stdin

```console
$ cat example.md | runblock
```

### With default command

```console
$ runblock -c "cat" example.md
```

### Watch mode

You can use the `--watch` flag to continuously monitor changes to your Markdown file and automatically re-run when the file is modified:

```console
$ runblock --watch example.md
```

This is useful during development as it allows you to see changes in real-time as you edit the Markdown file.

## How it works

`runblock` parses Markdown files and extracts fenced code blocks. Each code block can specify a command in the info string after the language identifier.

### Specifying commands in code blocks

Commands are specified after the language identifier, separated by a space:

    ```go /usr/bin/gofmt
    package main

    func main() {}
    ```

When `runblock` processes this block, it executes `/usr/bin/gofmt` with the code block content.

### Template variables

Commands support template variables using CEL (Common Expression Language) syntax:

| Variable | Description |
| --- | --- |
| `{{lang}}` | Language identifier of the code block |
| `{{content}}` | Content of the code block |
| `{{i}}` | Index of the code block (0-based) |

CEL expressions are supported within `{{ }}`:

```
# Ternary operator
{{ lang == "" ? "txt" : lang }}

# String concatenation
{{ "prefix_" + lang }}

# Arithmetic
{{ i + 1 }}
```

### Environment variables

The following environment variables are set when executing commands:

| Variable | Description |
| --- | --- |
| `CODEBLOCK_LANG` | Language identifier of the code block |
| `CODEBLOCK_CONTENT` | Content of the code block |
| `CODEBLOCK_INDEX` | Index of the code block (0-based) |

### Standard input

The code block content is also passed to the command via stdin.

## Examples

### Convert code blocks to images

    ```go /path/to/code2img {{lang}} -o output_{{i}}.png
    package main

    func main() {
        fmt.Println("Hello, World!")
    }
    ```

### Format code blocks

    ```go gofmt
    package main
    func main(){fmt.Println("hello")}
    ```

### Execute code blocks

    ```python python3
    print("Hello from Python")
    ```

### Using default command for all blocks

```console
$ runblock -c "cat > block_{{i}}.txt" example.md
```

This saves each code block to a separate file.

## Installation

**go install:**

```console
$ go install github.com/k1LoW/runblock@latest
```

## Flags

```
Flags:
  -c, --command string   default command for code blocks without explicit command
  -h, --help             help for runblock
  -v, --version          version for runblock
  -w, --watch            watch the file for changes and re-run on modifications
```
