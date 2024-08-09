# Go Kubernetes Downscaler - Helm Chart

This repository offers a Helm chart for the `go-kube-downscaler`.

## Important values

| Key                | Type   | Example                                                                                               | Description                                     |
|--------------------|--------|-------------------------------------------------------------------------------------------------------|-------------------------------------------------|
| image.tag          | string | `"23.2.0@sha256:4129e7e7551eb451ee2b43680ef818f3057304ad50888f79ec9722afab6c29ff"`                    | Tag of the image to use                         |
| arguments          | list   | `[--interval=60,--include-resources=deployments,statefulsets,horizontalpodautoscalers,scaledobjects]` | Arguments to pass to the kube-downscaler binary |
| excludedNamespaces | list   | `["namespace-a", "namespace-b"]`                                                                      | Namespaces to exclude from downscaling          |
| extraConfig        | string | `"DOWNSCALE_PERIOD: 'Mon-Sun 19:00-20:00 Europe/Berlin'"`                                              | Additional configuration in ConfigMap format    |

# Deploy go-kube-downscaler using Helm chart

This directory contains a tutorial to deploy go-kube-downscaler.

## Configuring your Deployment to downscale

Please add the annotations below based on your timezone on the deployments it should scale:

```yaml
metadata:
  annotations:
    downscaler/uptime: "Mon-Fri 07:00-19:00 US/Eastern"
```

## Quick Start

Below are instructions to quickly install and configure the go-kube-downscaler.

### Installing py-kube-downscaler

1. Make sure you're connected to the right cluster:

```bash
kubectl config current-context
```

2. Before deploying, make sure to update the **values.yaml** in the go-kube-downscaler chart depending on whether you want RBAC
   roles deployed or not:

```yaml
rbac:
  create: false
```

Note: In case RBAC is enabled, a new service account will be created for the go-kube-downscaler with certain privileges,
otherwise the 'default' one will be used.

3. Deploy the go-kube-downscaler:\
You can add our chart repository and deploy it by running:
```bash
helm repo add caas-team https://caas-team.github.io/helm-charts/

helm install go-kube-downscaler caas-team/go-kube-downscaler -n go-kube-downscaler
```

**OR**

You can alternatively clone this repository, change the current directory to the go-kube-downscaler repository and run:
```bash
helm install go-kube-downscaler ./chart -n go-kube-downscaler
```

4. Check the deployed release status:

```bash
helm list -n go-kube-downscaler
```

```
NAME                REVISION  UPDATED                   STATUS    CHART                     APP VERSION  NAMESPACE
go-kube-downscaler  1         Tue Sep 25 02:07:58 2018  DEPLOYED  go-kube-downscaler-0.5.1      0.5.1    go-kube-downscaler

```

5. Check whether go-kube-downscaler pod is up and running:

```bash
kubectl get pods -n go-kube-downscaler
```

```
NAME                                                     READY     STATUS    RESTARTS   AGE
go-kube-downscaler-go-kube-downscaler-7f58c6b5b7-rnglz   1/1       Running   0          6m
```

6. Check the Kubernetes event logs, to make sure the deployment of the go-kube-downscaler was successful:

```bash
kubectl get events -w
```

## Acknowledgments

Thanks to [Kube-downscaler](https://github.com/hjacobs/kube-downscaler) project authored
by [Henning Jacobs](https://github.com/hjacobs).