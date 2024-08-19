# GoKubeDownscaler

<div style="display: flex; gap: 1rem; flex-wrap: nowrap; overflow-x: auto;">
    <a href="/../../releases/" title="GitHub Release" style="flex-shrink: 0;"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/caas-team/GoKubeDownscaler?style=flat"></a>
    <a href="./LICENSE" title="GitHub License" style="flex-shrink: 0;"><img alt="GitHub License" src="https://img.shields.io/github/license/caas-team/GoKubeDownscaler?style=flat"></a>
    <a href="/../../graphs/contributors" title="Contributers" style="flex-shrink: 0;"><img alt="Contributers" src="https://img.shields.io/github/contributors/caas-team/GoKubeDownscaler?style=flat"></a>
    <a href="/../../stargazers" title="Stars" style="flex-shrink: 0;"><img alt="Stars" src="https://img.shields.io/github/stars/caas-team/GoKubeDownscaler?style=flat"></a>
</div>

A vertical autoscaler for Kubernetes workloads.
This is a golang port of the popular [(py-)kube-downscaler](github.com/caas-team/py-kube-downscaler) with improvements and quality of life changes.

## Table of contents

- [Use Cases](#use-cases)
- [Scalable Resources](#scalable-resources)
  - [Deployments](#deployments)
- [Concepts](#concepts)
  - [Layers](#layers)
  - [Values](#values)
- [Installation](#installation)
- [Configuration](#configuration)
  - [Arguments](#arguments)
  - [Annotations](#annotations)
- [Migrating from py-kube-downscaler](#migrating-from-py-kube-downscaler)
- [Differences to py-kube-downscaler](#differences-to-py-kube-downscaler)
- [Developing](#developing)
  - [Cloning the Repository](#cloning-the-repository)
  - [Setting up Pre-Commit](#setting-up-pre-commit)
  - [Testing the downscaler](#testing-the-downscaler)

## Use Cases

- Downscaling during non-work hours/times of day with low traffic

## Scalable Resources

These are the resources the Downscaler can scale:

<!-- Keep this list updated -->

- <span id="deployments">Deployments</span>:
  - sets the replica count to the downscale replicas

## Concepts

### Layers

<!-- TODO this -->

### Values

<!-- TODO this -->

## Installation

Installation is done via a [Helm Chart](./deployments/chart/README.md)

## Configuration

### Arguments

<!-- TODO this -->

### Annotations

<!-- TODO this -->

## Migrating from py-kube-downscaler

<!-- TODO this -->

## Differences to py-kube-downscaler

<!-- TODO this -->

## Developing

Please read the [contribution manifest](./CONTRIBUTING.md)

### Cloning the Repository

```bash
git clone https://github.com/caas-team/GoKubeDownscaler.git
cd GoKubeDownscaler
```

### Setting up Pre-Commit

```bash
brew install pre-commit
pre-commit install
brew install golangci-lint
brew install gofumpt
```

### Testing the downscaler

#### running the unit tests

```bash
go test -v --cover ./...
```

#### running the downscaler locally

The downscaler can be run locally by specifying a kubeconfig to use.
The kubeconfig should have at least the permissions as the Helm Charts [role.yaml](./deployments/chart/templates/role.yaml).
The downscaler will use the current-context in the kubeconfig.

```bash
go run -k=path/to/kubeconfig # ... additional configuration
```
