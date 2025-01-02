# GoKubeDownscaler

<a target="_blank" href="/../../releases/" title="GitHub Release"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/caas-team/GoKubeDownscaler?style=flat"></a>
<a target="_blank" href="./LICENSE" title="GitHub License"><img alt="GitHub License" src="https://img.shields.io/github/license/caas-team/GoKubeDownscaler?style=flat"></a>
<a target="_blank" href="/../../graphs/contributors" title="Contributors"><img alt="Contributors" src="https://img.shields.io/github/contributors/caas-team/GoKubeDownscaler?style=flat"></a>
<a target="_blank" href="/../../stargazers" title="Stars"><img alt="Stars" src="https://img.shields.io/github/stars/caas-team/GoKubeDownscaler?style=flat"></a>
<a target="_blank" href="https://communityinviter.com/apps/kube-downscaler/kube-downscaler" title="Slack Workspace"><img alt="Slack Workspace" src="https://img.shields.io/badge/slack-kube--downscaler-dark_green?style=flat&logo=slack"></a>

A horizontal autoscaler for Kubernetes workloads.
This is a golang port of the popular [(py-)kube-downscaler](https://github.com/caas-team/py-kube-downscaler) with improvements and quality of life changes.

## Documentation and Guides

The Documentation and Guides can be found on [our website](https://caas-team.github.io/GoKubeDownscaler).

An Offline copy of the documentation can be found in [`website/content/docs`](./website/content/docs) and [`website/content/guides`](./website/content/guides).
In there are Markdown files which can be viewed in any text editor or inside of a Markdown Viewer.

## Installation

Installation is done via the [Helm Chart](./deployments/chart/).
Information on how to install the Downscaler is on [our website](https://caas-team.github.io/GoKubeDownscaler/guides/getting-started/installation).

## Developing

This section covers the basics of developing on this repo, a more detailed guide can be found on [our website](https://caas-team.github.io/GoKubeDownscaler/guides/developing).

Please read the [contribution manifest](./CONTRIBUTING.md).

### Cloning the repository

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

### Testing the website

#### installing dependencies

```bash
npm install --prefix website
```

#### running it locally

```bash
npm run --prefix website start
```

after that the website is available on `localhost:3000/GoKubeDownscaler`
