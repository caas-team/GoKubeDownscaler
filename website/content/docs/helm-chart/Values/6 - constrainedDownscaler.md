---
title: constrainedDownscaler
id: constrainedDownscaler
---

# constrainedDownscaler

Sets the go-kube-downscaler to a constrained state if true, meaning that it does not run clusterwide but restricted to a list of namespaces. These namespaces are listed in [constrainedNamespaces](#constrainedNamespaces).

> ℹ️ **The default value for `constrainedDownscaler` is:**
>
> ```yaml
> constrainedDownscaler: false
> ```
