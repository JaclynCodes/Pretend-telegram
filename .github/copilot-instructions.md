# GitHub MCP Server - Coding Agent Instructions

## Project Overview

This is the **GitHub MCP Server**, an MCP (Model Context Protocol) server that connects AI tools directly to GitHub's platform. It enables AI agents to read repositories, manage issues/PRs, analyze code, and automate workflows through natural language interactions. The project is written in **Go 1.23.7+** and consists of approximately 7,500 lines of Go code, YAML, and Markdown files.

**Technology Stack:**
- Language: Go 1.23.7+
- Key Dependencies: github.com/google/go-github/v74, github.com/mark3labs/mcp-go, github.com/spf13/cobra, github.com/shurcooL/githubv4
- Build Tool: Go toolchain
- Container: Docker (distroless base image)
- Testing: Go's built-in testing with custom toolsnap validation

## Build, Test, and Validation Steps

### Prerequisites
**ALWAYS** ensure you have Go 1.23.7+ installed. The project uses `go.mod` to specify the exact Go version.

### Step-by-Step Build Process

1. **Download Dependencies** (takes ~5-10 seconds):
   ```bash
   go mod download
   ```

2. **Build the Server** (takes ~30-40 seconds):
   ```bash
   go build -v ./cmd/github-mcp-server
   ```
   This creates a `github-mcp-server` binary in the repository root.

3. **Run Tests** (takes ~50-60 seconds):
   ```bash
   script/test
   # Equivalent to: go test -race ./...
   ```
   
   **IMPORTANT**: If you modify tool definitions, you **MUST** update tool snapshots:
   ```bash
   UPDATE_TOOLSNAPS=true go test ./...
   ```
   Failing to do this will cause test failures with messages about "tool schema has changed unexpectedly".

4. **Run Linting** (takes ~15-20 seconds, first run may take longer to install golangci-lint):
   ```bash
   script/lint
   ```
   This runs `gofmt -s -w .` followed by `golangci-lint run` with version v1.60.1. The linter configuration is in `.golangci.yml`.

5. **Generate Documentation** (required after tool changes):
   ```bash
   go run ./cmd/github-mcp-server generate-docs
   # Or: ./github-mcp-server generate-docs
   ```
   This updates the README.md with current tool definitions. **ALWAYS** run this and commit changes if you modify tools.

### Complete Validation Workflow

To ensure your changes will pass CI, run these commands **in this order**:
```bash
go mod download                              # Download dependencies
script/lint                                  # Lint code (15-20s)
UPDATE_TOOLSNAPS=true go test ./...         # Update snapshots and test (50-60s)
go run ./cmd/github-mcp-server generate-docs # Update documentation
git diff README.md                           # Verify docs changed if tools modified
go build -v ./cmd/github-mcp-server         # Build binary (30-40s)
```

## Project Architecture and Layout

### Directory Structure

```
/
├── cmd/
│   ├── github-mcp-server/     # Main server entry point (main.go, generate_docs.go)
│   └── mcpcurl/               # CLI tool for testing MCP servers
├── pkg/
│   ├── github/                # Core GitHub tool implementations
│   │   ├── actions.go         # GitHub Actions workflows
│   │   ├── code_scanning.go   # CodeQL and code scanning
│   │   ├── context_tools.go   # User/team context
│   │   ├── dependabot.go      # Dependabot alerts
│   │   ├── discussions.go     # GitHub Discussions
│   │   ├── gists.go           # GitHub Gists
│   │   ├── issues.go          # Issues management
│   │   ├── pullrequests.go    # Pull requests
│   │   ├── repositories.go    # Repository operations
│   │   ├── search.go          # Search functionality
│   │   ├── secret_scanning.go # Secret scanning alerts
│   │   ├── tools.go           # Tool registration and grouping
│   │   └── __toolsnaps__/     # Tool schema snapshots for testing
│   ├── errors/                # Error handling utilities
│   ├── log/                   # Logging utilities
│   ├── raw/                   # Raw GitHub API client
│   ├── toolsets/              # Tool organization
│   └── translations/          # Internationalization
├── internal/
│   ├── ghmcp/                 # MCP server implementation
│   ├── githubv4mock/          # GraphQL mocking for tests
│   ├── toolsnaps/             # Tool snapshot testing framework
│   └── profiler/              # Performance profiling
├── script/
│   ├── test                   # Run tests with race detection
│   ├── lint                   # Run gofmt + golangci-lint
│   ├── generate-docs          # Generate README documentation
│   ├── licenses               # Generate third-party licenses
│   └── licenses-check         # Verify license compliance
├── .github/workflows/
│   ├── go.yml                 # Build and test on push/PR
│   ├── lint.yml               # Linting checks
│   ├── docs-check.yml         # Verify docs are up-to-date
│   ├── license-check.yml      # License compliance check
│   ├── codeql.yml             # CodeQL security scanning
│   └── code-scanning.yml      # Additional code scanning
├── go.mod                     # Go module definition
├── .golangci.yml              # Linter configuration
├── Dockerfile                 # Container build definition
└── README.md                  # Auto-generated documentation
```

### CI/CD Validation Pipeline

The following GitHub Actions workflows run on every push and PR:

1. **Build and Test** (`.github/workflows/go.yml`): Builds on ubuntu/windows/macos, runs `script/test`
2. **Linting** (`.github/workflows/lint.yml`): Runs golangci-lint v2.1
3. **Documentation Check** (`.github/workflows/docs-check.yml`): Verifies README.md is up-to-date with tool definitions
4. **License Check** (`.github/workflows/license-check.yml`): Verifies third-party license compliance
5. **CodeQL Analysis** (`.github/workflows/codeql.yml`): Security scanning for Go and GitHub Actions

### Key Files

- **Main Entry Point**: `cmd/github-mcp-server/main.go` - Cobra-based CLI with stdio and generate-docs commands
- **Tool Registration**: `pkg/github/tools.go` - Defines toolsets (actions, code_security, issues, pull_requests, etc.)
- **Linter Config**: `.golangci.yml` - Enables bodyclose, gocritic, gosec, makezero, misspell, nakedret, revive
- **Go Module**: `go.mod` - Specifies Go 1.23.7 and dependencies

### Tool Implementation Pattern

All tools follow this pattern:
1. Define in `pkg/github/<category>.go` (e.g., `code_scanning.go`)
2. Register in `pkg/github/tools.go` in appropriate toolset
3. Add tests in `pkg/github/<category>_test.go`
4. Tool schema snapshots stored in `pkg/github/__toolsnaps__/<tool_name>.snap`

### Common Pitfalls

1. **Forgetting to update toolsnaps**: Always run `UPDATE_TOOLSNAPS=true go test ./...` after modifying tool definitions
2. **Forgetting to regenerate docs**: Always run `go run ./cmd/github-mcp-server generate-docs` after modifying tools
3. **Race condition in tests**: The test suite uses `-race` flag; avoid shared mutable state in tests
4. **Build without dependencies**: Always run `go mod download` in a clean environment

### Running the Server

```bash
# Via stdio (requires GITHUB_PERSONAL_ACCESS_TOKEN env var)
export GITHUB_PERSONAL_ACCESS_TOKEN="your_token"
./github-mcp-server stdio

# Via Docker
docker run -i --rm -e GITHUB_PERSONAL_ACCESS_TOKEN ghcr.io/github/github-mcp-server
```

## Trust These Instructions

The information above has been validated by running all commands and reviewing all configuration files. Only search for additional information if you encounter errors not covered here or if these instructions are incomplete for your specific task.