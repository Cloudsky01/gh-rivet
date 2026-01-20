<div align="center">
  
  <img src="static/rivet_logo.png" alt="Rivet" width="180"/>
  
  # Rivet
  
  **A TUI for organizing and managing GitHub Actions workflows.**
  
  [![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://go.dev)
  [![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
  
  ---
  
</div>

---

## What is this?

A terminal UI for organizing GitHub Actions workflows into groups. Navigate workflows with vim keys, pin favorites, and avoid the messy default GitHub interface.

Wraps the [GitHub CLI (`gh`)](https://cli.github.com/) — uses your existing auth, no tokens needed.

## Requirements

* Go 1.21+
* [GitHub CLI (`gh`)](https://cli.github.com/) installed and authenticated.

## Installation

### Homebrew (macOS/Linux)
```bash
brew install cloudsky01/tap/rivet
```

### Go Install

```bash
go install github.com/Cloudsky01/gh-rivet/cmd/rivet@latest
```

### Build from Source

```bash
git clone https://github.com/Cloudsky01/gh-rivet
cd gh-rivet
make build
```

## Quick Start

**1. Initialize inside your repo:**
```bash
cd your-repo
rivet init  # Auto-detects repo, scans workflows, guides you through grouping
```

**2. Run:**
```bash
rivet  # Uses repository from .rivet.yaml
```

**Update repo later:**
```bash
rivet update-repo owner/repo
```

## Configuration

`rivet init` walks you through grouping workflows and choosing where to save the config.
Pick a user-specific config (`~/.config/rivet/config.yaml`) for personal prefs or save to `.github/.rivet.yaml` to share with your team.

### Configuration Precedence & Merging

Rivet loads configuration from multiple sources and merges them. The order of precedence (lowest to highest) is:

1.  **Repository Default**: `.github/.rivet.yaml` (Shared team defaults)
2.  **User Global**: `~/.config/rivet/config.yaml` (Your personal preferences)
3.  **Project User**: `.git/.rivet/config.yaml` (Your per-project overrides)

**Merging Logic:**
*   **Preferences**: Merged. You can set a global theme in your User Global config, and it will apply to all projects unless overridden.
*   **Groups**: Replaced. If a higher-precedence config defines `groups`, it completely replaces the groups from lower-precedence configs. This prevents messy merging of workflow lists.

**Example:**
```yaml
repository: owner/repo

groups:
  # Simple grouping
  - id: ci
    name: "CI/CD"
    workflows:
      - test.yml
      - build.yml

  # Nested grouping
  - id: services
    name: "Microservices"
    groups:
      - id: auth
        name: "Auth Service"
        workflows:
          - auth-test.yml
      - id: api
        name: "API Gateway"
        workflows:
          - api-deploy.yml

  # Custom display names & Pinning
  - id: infra
    name: "Infrastructure"
    workflowDefs:
      - file: terraform.yml
        name: "Terraform Apply (Prod)"
    pinnedWorkflows:
      - terraform.yml
```

## FAQ

**Does this require a GitHub Token?**
No. Uses your local `gh` CLI. If `gh run list` works, Rivet works.

**Why is it slow?**
Fetches live data from GitHub on demand. No aggressive caching = always fresh status.

**Works with GitHub Enterprise?**
Yes, if your `gh` CLI is authenticated to your instance.

## License

MIT
