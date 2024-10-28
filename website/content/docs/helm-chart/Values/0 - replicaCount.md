---
title: replicaCount
id: replicaCount
---

# replicaCount

`replicaCount` adjusts the number of replicas that the GoKubeDownscaler deployment will have.

> ℹ️ **The default value of `replicaCount` is:**
>
> ```yaml
> replicaCount: 1
> ```

Changing this to a different integer value will result in the deployment of the provided number of pods.

> ⚠️ **Warning**\
> This value should not be changed because multiple instances of the GoKubeDownscaler would be fighting about downscaling the targeted resources.
