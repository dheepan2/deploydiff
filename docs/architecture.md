# DeployDiff Architecture

DeployDiff is a deployment intelligence engine for Kubernetes.

The project is built as a layered architecture.

CLI
    ↓
Manifest Loader
    ↓
Parser
    ↓
Knowledge Graph
    ↓
Diff Engine
    ↓
Risk Engine
    ↓
Reporting

Each layer has a single responsibility and is independently testable.