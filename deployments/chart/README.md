# go kubernetes Downscaler - Helm Chart

This repository provides a Helm chart for the `go-kube-downscaler`.\
Here you can find a brief overview of the charts components and a quick installation guide.

- [Components](#components)
  - [Deployment](#deployment)
  - [Serviceaccount](#serviceaccount)
  - [Configmap](#configmap)
  - [Clusterrole](#clusterrole)
  - [Role](#role)
- [Values](#values)
  - [replicaCount](#replicaCount)
  - [image](#image)
  - [arguments](#arguments)
  - [includedResources](#includedResources)
  - [nameOverride](#nameOverride)
  - [fullnameOverride](#fullnameOverride)
  - [constrainedDownscaler](#constrainedDownscaler)
  - [constrainedNamespaces](#constrainedNamespaces)
  - [podSecurityContext](#podSecurityContext)
  - [securityContext](#securityContext)
- [Installation](#installation)
  - [Prerequisits](#prerequisits)
  - [Install the Chart](#install-the-chart)
- [Configure scaling for specific Workloads](#configure-scaling-for-specific-workloads)

## Components

### Deployment

The deployment.yaml file creates the main Deployment of the go-kube-downscaler with a reference to the provided ServiceAccount and ConfigMap.

### Serviceaccount

The serviceaccount.yaml file creates a ServiceAccount that will be used by the go-kube-downscaler to interact with Kubernetes.

### Configmap

The configmap.yaml file creates a new ConfigMap with the provided name. It is used to store environment variables like `EXCLUDE_NAMESPACES`.

### Clusterrole

If [constrainedDownscaler](#constrainedDownscaler) is set to false the clusterrole.yaml file creates a ClusterRole and a ClusterRoleBinding which allow the actions the go-kube-downscaler needs to scale down workloads in a clusterwide context.

### Role

If [constrainedDownscaler](#constrainedDownscaler) is set to true the role.yaml file creates a Role and a RoleBinding for every namespace listed in [constrainedNamespaces](#constrainedNamespaces) in order to allow the necessary actions to scale down workloads.

## Values

Here is a brief overview of all the values contained in this chart:

<!-- It's recommended to disable word wrap to view/edit this table -->

| **Key**                                                                 | **Description**                                                                                                                                                                                                              |
| ----------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| <a target="_blank" id="replicaCount">replicaCount</a>                   | Adjusts the number of replicas that the go-kube-downscaler deployment will have.                                                                                                                                             |
| <a target="_blank" id="image">image</a>                                 | Contains the repository URL and the image tag for the image you want to use.                                                                                                                                                 |
| <a target="_blank" id="arguments">arguments</a>                         | Defines command line arguments for the container.                                                                                                                                                                            |
| <a target="_blank" id="includedResources">includedResources</a>         | Defines all the resources that the go-kube-downscaler is supposed to be able to handle.                                                                                                                                      |
| <a target="_blank" id="fullnameOverride">fullnameOverride</a>           | Overrides the full names of the created Kubernetes resources with the provided name if set.                                                                                                                                  |
| <a target="_blank" id="nameOverride">nameOverride</a>                   | Overrides the chart name for the created Kubernetes resources.                                                                                                                                                               |
| <a target="_blank" id="constrainedDownscaler">constrainedDownscaler</a> | Sets the go-kube-downscaler to a constrained state if true, meaning that it does not run clusterwide but restricted to a list of namespaces. These namespaces are listed in [constrainedNamespaces](#constrainedNamespaces). |
| <a target="_blank" id="constrainedNamespaces">constrainedNamespaces</a> | A list of namespaces the go-kube-downscaler is going to operate on when constrained.                                                                                                                                         |
| <a target="_blank" id="serviceAccount">serviceAccount</a>               | Specifies whether a new ServiceAccount is supposed to be created and if not, what the name of the ServiceAccount to use is.                                                                                                  |
| <a target="_blank" id="podSecurityContext">podSecurityContext</a>       | Defines the SecurityContext on pod level. (More info [here](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/))                                                                                     |
| <a target="_blank" id="securityContext">securityContext</a>             | Defines the SecurityContext on container level. (More info [here](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/))                                                                               |
| <a target="_blank" id="resources">resources</a>                         | Defines the CPU and memory limits and requests.                                                                                                                                                                              |
| <a target="_blank" id="nodeSelector">nodeSelector</a>                   | Defines labels of nodes that you want the deployment to schedule them on.                                                                                                                                                    |
| <a target="_blank" id="tolerations">tolerations</a>                     | Adds tolerations to the pods of the deployment to be able to be scheduled on nodes with matching taints. (More info [here](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/))                   |
| <a target="_blank" id="affinity">affinity</a>                           | Defines rules for node affinity and pod affinity/anti-affinity. (More info [here](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/))                                                                 |
| <a target="_blank" id="configMapName">configMapName</a>                 | The name of the configmap for the go-kube-downscaler.                                                                                                                                                                        |
| <a target="_blank" id="excludedNamespaces">excludedNamespaces</a>       | A list of namespaces that are supposed to be excluded from the downscaling process of the go-kube-downscaler.                                                                                                                |
| <a target="_blank" id="extraConfig">extraConfig</a>                     | Adds additional specified environment variables to the ConfigMap.                                                                                                                                                            |

The default values can be found [here](./values.yaml).

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
