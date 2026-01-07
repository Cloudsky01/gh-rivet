<div align="center">
  <img src="rivet.jpg" alt="Rivet" width="200"/>

  # Rivet
  
  **A TUI for organizing and managing GitHub Actions workflows.**

  [![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://go.dev)
  [![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
</div>

---

## What is this?

If you work in a repository with dozens of workflows, the default GitHub UI and `gh run list` are messy. 

**Rivet** wraps the [GitHub CLI (`gh`)](https://cli.github.com/) to provide a structured, hierarchical view of your actions. It lets you group workflows (e.g., by Team or Environment) and navigate them quickly without leaving your terminal.

## Why use it?

* **Organize the noise:** Turn a flat list of 50+ YAML files into structured folders.
* **No new auth:** It uses your existing `gh` CLI credentials. No tokens to manage.
* **Pinning:** Keep your most frequently used workflows at the top.
* **Vim keys:** Navigate `j`/`k` without reaching for the mouse.

## Requirements

* Go 1.21+
* [GitHub CLI (`gh`)](https://cli.github.com/) installed and authenticated.

## Installation

### Go Install
```bash
go install github.com/Cloudsky01/gh-rivet/cmd/rivet@latest
```

### From Source
```bash
git clone https://github.com/Cloudsky01/gh-rivet
cd rivet
make build
```

## Usage

### Initialize config
```bash
# Run inside your repository
rivet init
```

### Run
```bash
rivet -r owner/repo
```

## Configuration

Rivet looks for a `.rivet.yaml` file in the directory where you run it. You should commit this file to your repository so your team shares the same structure.

### Example .rivet.yaml:
```yaml
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

### Does this require a GitHub Token?

No. It shells out to your local `gh` executable. If you can run `gh run list` in your terminal, Rivet works.

### Why is it slow?

Rivet fetches data from GitHub on demand. It does not cache aggressively to ensure you see the latest run status. If the GitHub API is slow, the TUI will wait.

### Does it work with GitHub Enterprise?

Yes, as long as your `gh` CLI is authenticated against your Enterprise instance.

## License

MIT
