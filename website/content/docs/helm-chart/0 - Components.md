---
title: Components
id: components
---

# Components

The GoKubeDownscaler Helm Chart can create the following Kubernetes Objects:

## Deployment

The [deployment.yaml](https://github.com/caas-team/GoKubeDownscaler/blob/main/deployments/chart/templates/deployment.yaml) file creates the main Deployment of the GoKubeDownscaler with a reference to the provided ServiceAccount and ConfigMap.

## Serviceaccount

The [serviceaccount.yaml](https://github.com/caas-team/GoKubeDownscaler/blob/main/deployments/chart/templates/serviceaccount.yaml) file creates a ServiceAccount that will be used by the go-kube-downscaler to interact with Kubernetes.

## Configmap

The [configmap.yaml](https://github.com/caas-team/GoKubeDownscaler/blob/main/deployments/chart/templates/configmap.yaml) file creates a new ConfigMap with the provided name. It is used to store environment variables like `EXCLUDE_NAMESPACES`.

## Clusterrole

The [clusterrole.yaml](https://github.com/caas-team/GoKubeDownscaler/blob/main/deployments/chart/templates/clusterrole.yaml) file creates a ClusterRole and a ClusterRoleBinding for the desired cluster if [constrainedDownscaler](http://localhost:3000/GoKubeDownscaler/docs/helm-chart/Values) is set to false. These allow the actions the GoKubeDownscaler needs to scale down workloads in a clusterwide context.

## Role

The [role.yaml](https://github.com/caas-team/GoKubeDownscaler/blob/main/deployments/chart/templates/role.yaml) file creates a Role and a RoleBinding for every namespace listed in [constrainedNamespaces](http://localhost:3000/GoKubeDownscaler/docs/helm-chart/Values) if [constrainedDownscaler](http://localhost:3000/GoKubeDownscaler/docs/helm-chart/Values) is set to true.
