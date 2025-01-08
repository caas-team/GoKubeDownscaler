# go kubernetes Downscaler - Helm Chart

This is the official Helm Chart for the `GoKubeDownscaler`.

In order to installing the GoKubeDownscaler using our Helm Chart you only need to have Helm installed on a system and have access to a Kubernetes cluster in some kind of way.

## Installation

The installation is quite simple.

First you have to add our chart repository by running:

```bash
helm repo add caas-team https://caas-team.github.io/helm-charts/
```

This will add all of our published Helm Charts to a local Helm repo named caas-team.

After that you just need to run the following command:

```bash
helm install go-kube-downscaler caas-team/go-kube-downscaler
```

## Customizing your installation

You can change a lot about your installation of the GoKubeDownscaler.

You can find information on how to do that on our [documentation page](https://caas-team.github.io/GoKubeDownscaler/).
