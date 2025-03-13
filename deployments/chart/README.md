# Go Kubernetes Downscaler - Helm Chart

This is the official Helm Chart for the `GoKubeDownscaler`.

In order to install the GoKubeDownscaler using our Helm Chart
you only need to have Helm installed on a system and have access to a Kubernetes cluster in some kind of way.

## Installation

The installation is quite simple.

All you have to is install our chart from the GitHub registry by running:

```bash
helm upgrade -i gokubedownscaler oci://ghcr.io/caas-team/charts/go-kube-downscaler
```

You can also alternatively add our helm chart repo in order to install the chart.

```bash
helm repo add caas-team https://caas-team.github.io/helm-charts/
```

This will add all of our published Helm Charts to a local Helm repo named caas-team.

After that you just need to run the following command:

```bash
helm install go-kube-downscaler caas-team/go-kube-downscaler
```

## Customizing Your Installation

Our Helm Chart offers a lot of customizable values for your installation of the GoKubeDownscaler.

You can find information on how to adjust the chart to your needs on our [documentation page](https://caas-team.github.io/GoKubeDownscaler/).
