# minibp

A minimal Android.bp (Blueprint) parser and Ninja build file generator written in Go.

## Features

- **Full Soong-style build rules**: Supports 20+ module types including:
  - C/C++: `cc_library`, `cc_library_static`, `cc_library_shared`, `cc_object`, `cc_binary`, `cpp_library`, `cpp_binary`
  - Go: `go_library`, `go_binary`, `go_test`
  - Java: `java_library`, `java_library_static`, `java_library_host`, `java_binary`, `java_binary_host`, `java_test`, `java_import`
  - Proto: `proto_library`, `proto_gen`
  - Other: `filegroup`, `custom`

- **Soong syntax support**:
  - `defaults` modules for property reuse
  - `package` modules for package-level defaults
  - `soong_namespace` for namespace definitions
  - Module references: `:module` and `:module{.tag}` syntax
  - Visibility control: `//visibility:public`, `//visibility:private`, etc.
  - Select statements for conditional compilation

- **Desc comments**: Generate Soong-style build descriptions
  - Format: `//<source_dir>:<module_name> <action> <src_file>`

- **Transitive header includes**: Option B style - if A depends on B, and B depends on C, A automatically includes C's headers

- **Wildcard support**: `filegroup` supports `**` recursive glob patterns

- **Custom commands**: Full support for `$in` and `$out` variables in custom rules

- **Duplicate rule handling**: Avoids duplicate ninja rule definitions

## Usage

```bash
# Parse a single .bp file
go run ./cmd/minibp/main.go Android.bp

# Parse all .bp files in a directory
go run ./cmd/minibp/main.go -a .

# Specify output file
go run ./cmd/minibp/main.go -o build.ninja Android.bp
```

## Example

```bash
cd examples
go run ../cmd/minibp/main.go -a .
ninja
```

## Project Structure



```

minibp/

├── cmd/minibp/          # CLI entry point

├── dag/                 # DAG module registry

├── examples/            # Example build files

├── ninja/               # Ninja generator & rules

│   ├── gen.go          # Build file generation

│   ├── rules.go        # Build rule interfaces and utilities

│   ├── writer.go       # Ninja output writer

│   ├── cc.go           # C/C++ rules (cc_library, cc_binary, etc.)

│   ├── go.go           # Go rules (go_library, go_binary, etc.)

│   ├── java.go         # Java rules (java_library, java_binary, etc.)

│   ├── filegroup.go    # File group rules

│   ├── custom.go       # Custom and proto rules

│   └── defaults.go     # Defaults, package, soong_namespace rules

├── parser/             # Blueprint parser

│   ├── ast.go         # AST definitions

│   ├── lexer.go       # Tokenizer

│   └── parser.go      # Parser

└── module/            # Module registry

```

## Ninja Output Example

```
# //examples:libutil gcc util.c
# //examples:libutil gcc helper.c
build libutil_util.o: cc_compile util.c
 flags = -Wall -O2
build libutil_helper.o: cc_compile helper.c
 flags = -Wall -O2
build liblibutil.a: cc_archive libutil_util.o libutil_helper.o
```

## Building

```bash
go build ./cmd/minibp
```

## Testing

```bash
go test ./...
```