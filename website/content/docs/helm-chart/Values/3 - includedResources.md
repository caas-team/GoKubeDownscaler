---
title: includedResources
id: includedResources
---

# includedResources

`includedResources` defines all the resource types that the go-kube-downscaler is supposed to be able to handle.

> ℹ️ **The default value for `includedResources` is:**
>
> ```yaml
> includedResources:
>   - deployments
> ```

Supported resources are:

## Deployments

## Statefulsets

## Daemonsets

## Rollouts

## HorizontalPodAutoscalers

## Jobs

## Cronjobs

## ScaledObjects

## Stacks

## PodDisruptionBudgets

## Prometheuses

```yaml
includedResources:
  - deployments
  - statefulsets
  - daemonsets
  - rollouts
  - horizontalpodautoscalers
  - jobs
  - cronjobs
  - scaledobjects
  - stacks
  - poddisruptionbudgets
  - prometheuses
```

You can also find these as comments in the [values.yaml](https://github.com/caas-team/GoKubeDownscaler/blob/main/deployments/chart/values.yaml#L10-L21).
