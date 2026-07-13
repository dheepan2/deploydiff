# DeployDiff

> **Understand the deployment impact of your Kubernetes changes before you merge.**

DeployDiff is an open-source CLI that analyzes Kubernetes deployment manifests and explains **what your changes mean**, not just **what changed**.

Instead of comparing YAML files line by line, DeployDiff builds a dependency graph from your Kubernetes resources and provides deployment-focused insights such as:

* Which resources were added, removed, or modified
* New or removed deployment dependencies
* Potential rollout risks
* Deployment order changes
* Blast radius analysis
* Human-readable deployment summaries

---

# Why DeployDiff?

Traditional Git diffs answer:

> What changed?

DeployDiff answers:

* What will be deployed?
* What depends on what?
* Is this deployment safe?
* What services are affected?
* Is a Secret or ConfigMap missing?
* Has the deployment order changed?
* What is the operational impact of this pull request?

---

# Example

Instead of seeing:

```diff
- replicas: 3
+ replicas: 5

- image: customer:v1.2
+ image: customer:v1.3
```

DeployDiff shows:

```text
Deployment Impact
────────────────────────────────────

Resources Modified
• Deployment customer-api

Changes
• Image: customer:v1.2 → customer:v1.3
• Replicas: 3 → 5

Dependencies Added
• customer-api → payment-secret

Risk
MEDIUM

Reason
Deployment now depends on a new Secret that must exist before rollout.
```

---

# Features

## Manifest Analysis

* Kubernetes manifest parsing
* Helm chart support
* Skaffold support
* Kustomize support (planned)

## Deployment Graph

Automatically discovers dependencies between resources including:

* Deployment
* StatefulSet
* DaemonSet
* CronJob
* Service
* Ingress
* ConfigMap
* Secret
* ServiceAccount
* PersistentVolumeClaim
* HorizontalPodAutoscaler
* NetworkPolicy

No annotations required.

---

## Deployment Diff

Compare any two deployment states.

Examples:

Git-reference comparison is planned:

```bash
deploydiff compare --base origin/main --head HEAD
```

```bash
deploydiff compare ./release-v1 ./release-v2
```

```bash
deploydiff compare ./before ./after
```

---

## Insights

DeployDiff aims to detect:

* Added dependencies
* Removed dependencies
* Missing Secrets
* Missing ConfigMaps
* Circular dependencies
* Deployment order changes
* Blast radius
* Resource ownership
* Rollout risks

---

# Roadmap

## v0.1 — Foundation

* CLI
* Manifest parser
* Resource model
* Dependency graph
* Console output

## v0.2 — Deployment Diff

* Graph comparison
* Resource changes
* Dependency changes
* Risk analysis
* Markdown report

## v0.3 — Reports

* HTML reports
* Graph export (SVG)
* JSON output
* CI-friendly output

## v0.4 — GitHub Integration

* GitHub Action
* Pull request summary
* Markdown comments
* SARIF output (planned)

## v0.5 — Intelligence

* Blast radius analysis
* Deployment order validation
* Missing dependency detection
* Rollout simulation

## v1.0

* Stable CLI
* Plugin architecture
* Comprehensive documentation
* Homebrew installation
* Docker image

---

# Long-Term Vision

DeployDiff is designed to become a deployment intelligence platform for Kubernetes.

Future capabilities include:

* Live cluster comparison
* Drift detection
* Deployment history
* AI-assisted explanations
* Deployment simulation
* Argo CD and Flux integrations
* Multi-cluster analysis

---

# Project Principles

* CLI-first
* GitOps-friendly
* Kubernetes-native
* No cluster required for core analysis
* Fast and deterministic
* Extensible through plugins
* CI/CD friendly

---

# Tech Stack

* Go
* Cobra
* client-go
* Helm SDK
* yaml.v3
* Graphviz (optional)
* GitHub Actions

---

# Contributing

DeployDiff is in its early stages, and contributions are welcome.

Areas where help is especially appreciated:

* Kubernetes resource parsers
* Dependency extraction
* Graph algorithms
* Report generation
* Documentation
* GitHub integrations
* Testing

---

# License

Apache 2.0
