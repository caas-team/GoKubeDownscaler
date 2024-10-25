---
title: Installation
id: installation
---

We provide a Helm Chart for an easy installation of the GoKubeDownscaler.

## Prerequisits
In order to get started installing the GoKubeDownscaler using our Helm Chart you need:
- Helm installed on a system
- have access to a Kubernetes cluster in some kind of way.

You can find out how to install Helm [here](https://helm.sh/docs/intro/).

Also make sure your context is set to the right cluster and namespace where you want to deploy the GoKubeDownscaler.

## Install the chart

In order to install the chart you only need to add our Helm Chart repository by running:

```bash
helm repo add caas-team https://caas-team.github.io/helm-charts/
```

Followed by the actual install command:
```bash
helm install go-kube-downscaler caas-team/go-kube-downscaler
```

Now the GoKubeDownscaler should be properly installed.

You can find out how to customize your installation of the GoKubeDownscaler [here](http://localhost:3000/GoKubeDownscaler/docs/helm-chart/Values).
