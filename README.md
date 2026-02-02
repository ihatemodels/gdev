# gdev

A TUI-based development workflow manager that wraps git, GitHub, and Claude Code into a unified interface.

## Features

- **Branch Management** - Create, switch, and manage git branches from the TUI
- **Pull Requests** - Open and manage PRs directly from the interface
- **Claude Code Integration** - Start, monitor, and manage Claude Code sessions across branches
- **Session Management** - Track multiple Claude Code sessions working on different tasks
- **Unified Workflow** - One tool to orchestrate your entire development flow

## Installation

```bash
make build
```

Or with a specific version:

```bash
make build VERSION=1.0.0
```

## Usage

```bash
./gdev
```

## Requirements

- Go 1.21+
- git
- gh (GitHub CLI)
- claude (Claude Code CLI)

## License

MIT
