# Troubleshooting

- [Synchronous operation](#synchronous-operation)

## Synchronous operation

Error:

```
Operation cannot be fulfilled on xxxxxxx.xxxxx \"xxxxxxxxxxx\": the object has been modified;   please apply your changes to the latest version and try again
```

Causes:

- running multiple downscalers on the same resources
- the resource was modified while the resource was scaled

Fixes:

- do not run multiple downscalers on the same resources
- the `--max-retries-on-conflict` argument enables users to specify the number of retries for the downscaler when a conflict occurs. While the affected resource will likely be scaled in the next cycle without this optional argument, it is highly recommended to use it in conjunction with the `--once` argument

> [!Note]
> this is a pretty unavoidable issue due to there being no easy way to lock the resource from being edited while the downscaler is scaling it. The py-kube-downscaler solved this by just overwriting the changes made during scaling
