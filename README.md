<div align="center">
  <img src="static/rivet_logo.png" alt="Rivet" width="200"/>

  # Rivet
  
  **A TUI for organizing and managing GitHub Actions workflows.**

  [![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://go.dev)
  [![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
</div>

---

## What is this?

A terminal UI for organizing GitHub Actions workflows into groups. Navigate workflows with vim keys, pin favorites, and avoid the messy default GitHub interface.

Wraps the [GitHub CLI (`gh`)](https://cli.github.com/) — uses your existing auth, no tokens needed.

## Requirements

* Go 1.21+
* [GitHub CLI (`gh`)](https://cli.github.com/) installed and authenticated.

## Installation

```bash
go install github.com/Cloudsky01/gh-rivet/cmd/rivet@latest
```

Or build from source:
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

`rivet init` creates a `.rivet.yaml` file. Commit it to share the structure with your team.

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
