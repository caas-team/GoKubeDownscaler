# GoKubeDownscaler

<!-- style block because inline styles will be sanitized -->
<style>
  .scrollableBox {
    display: flex;
    gap: 1rem;
    flex-wrap: nowrap;
    overflow-x: auto;
  }
  .scrollableItem {
    display: flex;
    gap: 1rem;
    flex-wrap: nowrap;
    overflow-x: auto;
  }
</style>

<div class="scrollableBox">
    <a href="/../../releases/" title="GitHub Release" class="scrollableItem"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/caas-team/GoKubeDownscaler?style=flat"></a>
    <a href="./LICENSE" title="GitHub License" class="scrollableItem"><img alt="GitHub License" src="https://img.shields.io/github/license/caas-team/GoKubeDownscaler?style=flat"></a>
    <a href="/../../graphs/contributors" title="Contributers" class="scrollableItem"><img alt="Contributers" src="https://img.shields.io/github/contributors/caas-team/GoKubeDownscaler?style=flat"></a>
    <a href="/../../stargazers" title="Stars" class="scrollableItem"><img alt="Stars" src="https://img.shields.io/github/stars/caas-team/GoKubeDownscaler?style=flat"></a>
</div>

A vertical autoscaler for Kubernetes workloads.
This is a golang port of the popular [(py-)kube-downscaler](github.com/caas-team/py-kube-downscaler) with improvements and quality of life changes.

<!-- Don't use any heading or html tag with id smaller/deeper than h4(####) -->

## Table of contents

<!-- Keep this updated -->
<!-- Do add any heading bigger than h4 (####). If it shouldn't be on the list, it shouldn't be a heading -->
<!-- Do not add h4 or smaller in this list. If it should be on here but is h4 or smaller, you should restructure -->

- [Scalable Resources](#scalable-resources)
  - [Deployments](#deployments)
- [Installation](#installation)
- [Configuration](#configuration)
  - [Annotations](#annotations)
  - [Arguments](#arguments)
  - [Environment Variables](#environment-variables)
  - [Timespans](#timespans)
  - [Duration](#duration)
- [Concepts](#concepts)
  - [Layers](#layers)
  - [Values](#values)
- [Migrating from py-kube-downscaler](#migrating-from-py-kube-downscaler)
  - [Basic Migration](#basic-migration)
  - [Edge Cases](#edge-cases)
  - [Differences to py-kube-downscaler](#differences-to-py-kube-downscaler)
- [Developing](#developing)
  - [Cloning the Repository](#cloning-the-repository)
  - [Setting up Pre-Commit](#setting-up-pre-commit)
  - [Testing the downscaler](#testing-the-downscaler)

## Scalable Resources

These are the resources the Downscaler can scale:

<!-- Keep this list updated as more scalable resources are implemented -->

- <span id="deployments">Deployments</span>:
  - sets the replica count to the downscale replicas

<!-- resources with known downscale behaviour: (uncomment once implemented)
- <span id="horizontal-pod-autoscalers">Horizontal Pod Autoscalers (HPA)</span>:
  - sets the minReplicas of the HPA to the [downscale replicas](#downscale-replicas). will throw an error if the downscale replicas is >1
- <span id="cronjobs">CronJobs</span>:
  - sets the cronjobs suspend property to true
-->

## Installation

Installation is done via the [Helm Chart](./deployments/chart/README.md)

## Configuration

### Annotations

Annotations can be applied to the workload or the namespace. See the [layers concept](#layers) for more details on which of the layers [values](#values) will be used.

- <span id="downscaler/downscale-period">downscaler/downscale-period</span>:
  - sets the [downscale-period](#downscale-period) value on the [workload](#workload-layer) or [namespace](#namespace-layer) layer
- <span id="downscaler/downtime">downscaler/downtime</span>:
  - sets the [downtime](#downtime) value on the [workload](#workload-layer) or [namespace](#namespace-layer) layer
- <span id="downscaler/upscale-period">downscaler/upscale-period</span>:
  - sets the [upscale-period](#upscale-period) value on the [workload](#workload-layer) or [namespace](#namespace-layer) layer
- <span id="downscaler/uptime">downscaler/uptime</span>:
  - sets the [uptime](#uptime) value on the [workload](#workload-layer) or [namespace](#namespace-layer) layer
- <span id="downscaler/upscale-period">downscaler/upscale-period</span>:
  - sets the [upscale-period](#upscale-period) value on the [workload](#workload-layer) or [namespace](#namespace-layer) layer
- <span id="downscaler/exclude">downscaler/exclude</span>:
  - sets the [exclude](#exclude) value on the [workload](#workload-layer) or [namespace](#namespace-layer) layer
- <span id="downscaler/exclude-until">downscaler/exclude-until</span>:
  - sets the [exclude-until](#exclude-until) value on the [workload](#workload-layer) or [namespace](#namespace-layer) layer
- <span id="downscaler/force-uptime">downscaler/force-uptime</span>:
  - sets the [force-uptime](#force-uptime) value on the [workload](#workload-layer) or [namespace](#namespace-layer) layer
- <span id="downscaler/force-downtime">downscaler/force-downtime</span>:
  - sets the [force-downtime](#force-downtime) value on the [workload](#workload-layer) or [namespace](#namespace-layer) layer
- <span id="downscaler/downscale-replicas">downscaler/downscale-replicas</span>:
  - sets the [downscale-replicas](#downscale-replicas) value on the [workload](#workload-layer) or [namespace](#namespace-layer) layer
- <span id="downscaler/grace-period">downscaler/grace-period</span>:
  - sets the [grace-period](#grace-period) value on the [workload](#workload-layer) or [namespace](#namespace-layer) layer

### Arguments

CLI arguements set [layer values](#values) and runtime configuration at the start of the program. See the [layers concept](#layers) for more details on which of the layers [values](#values) will be used.

Layer Values:

- <span id="--upscale-period">--upscale-period</span>:
  - sets the [upscale-period](#upscale-period) value on the [cli layer](#cli-layer)
- <span id="--default-uptime">--default-uptime</span>:
  - sets the [default-uptime](#default-uptime) value on the [cli layer](#cli-layer)
- <span id="--downscale-period">--downscale-period</span>:
  - sets the [downscale-period](#downscale-period) value on the [cli layer](#cli-layer)
- <span id="--default-downtime">--default-downtime</span>:
  - sets the [default-downtime](#default-downtime) value on the [cli layer](#cli-layer)
- <span id="--downtime-replicas">--downtime-replicas</span>:
  - sets the [downscale replicas](#downscale-replicas) value on the [cli layer](#cli-layer)
- <span id="--explicit-include">--explicit-include</span>:
  - sets the [exclude value](#exclude) on the [cli layer](#cli-layer) to true, which excludes every workload unless the exclude value on the [workload](#workload-layer) or [namespace](#namespace-layer) layer is set to false. See the [layers concept](#layers) for more details.

Runtime Configuration:

- <span id="--dry-run">--dry-run</span>:
  - boolean
  - sets the downscaler into dry run mode, which makes it just print what it would have done, instead of actually doing it
  - default: false
- <span id="--debug">--debug</span>:
  - boolean
  - makes the downscaler print debug information
  - default: false
- <span id="--once">--once</span>:
  - boolean
  - makes the downscaler exit after one scan
  - default: false
- <span id="--interval">--interval</span>:
  - [duration](#duration)
  - sets the time between scans
  - default: 30s
- <span id="--namespace">--namespace</span>:
  - comma seperated list of namespaces (`some-ns,other-ns` or `some-ns, other-ns`)
  - makes the downscaler only get the specified namespaces
  - default: all namespaces
- <span id="--include-resources">--include-resources</span>:
  - comma seperated list of [scalable resources](#scalable-resources) (`deployments,statefulsets` or `deployments, statefulsets`)
  - restricts the downscaler to only scale workloads of this resource kind
  - default: deployments
- <span id="--exclude-namespaces">--exclude-namespaces</span>:
  - comma seperated list of namespaces (`some-ns,other-ns` or `some-ns, other-ns`)
  - makes the downscaler exclude the specified namespaces
  - default: kube-system, kube-downscaler
- <span id="--matching-labels">--matching-labels</span>:
  - comma seperated list of labels with their value (`some-label=val,other-label=value` or `some-label=val, other-label=value`)
  - makes the downscaler only include workloads which have any label that machtes any of the specified labels and values
  - default: none
- <span id="time-annotation">time-annotation</span>:
  - key of annotation with an RFC3339 Timestamp
  - when set grace-period will use the timestamp in the annotation instead of the creation time of the workload
  - default: none (uses the workloads creation time)

### Environment Variables

Environment Variables set [layer values](#values) and runtime configuration at the start of the program. See the [layers concept](#layers) for more details on which of the layers [values](#values) will be used.

Layer Values:

- <span id="upscale-period-env">UPSCALE_PERIOD</span>:
  - sets the [upscale-period](#upscale-period) value on the [env layer](#env-layer)
- <span id="uptime-env">DEFAULT_UPTIME</span>:
  - sets the [uptime](#uptime) value on the [env layer](#env-layer)
- <span id="downscale-period-env">DOWNSCALE_PERIOD</span>:
  - sets the [downscale-period](#downscale-period) value on the [env layer](#env-layer)
- <span id="downtime-env">DEFAULT_DOWNTIME</span>:
  - sets the [downtime](#downtime) value on the [env layer](#env-layer)

Runtime Configuration:

- <span id="exclude-namespaces-env">EXCLUDE_NAMESPACES</span>:
  - overwrites the value set by the [--exclude-namespaces](#--exclude-namespaces) cli argument
- <span id="exclude-deployments-env">EXCLUDE_DEPLOYMENTS</span>:
  - overwrites the value set by the [--exclude-deployments](#--exclude-deployments) cli argument

### Timespans

There are two different kinds of Timespans.

- Absolute Timespans: a timespan defined by two RFC3339 timestamps
- Relative Timespans: reoccuring on a schedule

#### Configuration of an Absolute Timespan

```text
<RFC3339-Timestamp>-<RFC3339-Timestamp>
or
<RFC3339-Timestamp> - <RFC3339-Timestamp>
```

eg.: `2024-07-29T08:30:00Z - 2024-07-29T16:00:00+02:00`

#### Configuration of a Relative Timespan

```text
<Weekday-From>-<Weekday-To> <Time-Of-Day-From>-<Time-Of-Day-To> <Timezone>
```

eg.:

```text
Mon-Fri 08:00-20:00 Europe/Berlin    # From Monday to Friday: from 08:00 to 20:00
Sat-Sun 00:00-24:00 UTC              # On The Weekend: the entire day
Mon-Fri 20:00-08:00 PST              # From Monday to Friday: from Midnight to 08:00 and from 20:00 until end of day
Mon-Sun 00:00-00:00 America/New_York # The timespan never matches, this would not do anything
Mon-Tue 20:00-24:00 CEST             # On Monday and Tuesday: from 20:00 to midnight
Mon-Tue 20:00-00:00 Europe/Amsterdam # On Monday and Tuesday: from 20:00 to midnight
```

Valid Values:

- Weekdays: (case-insensitive)
  - Mon
  - Tue
  - Wed
  - Thu
  - Fri
  - Sat
  - Sun
- Timezones: all from the [IANA Time Zone database](https://www.iana.org/time-zones)
- Time of day: 00:00 - 24:00

#### Multiple/Complex Timespans

In some cases you need to define multiple Timespans. You can do this like this:

```
<TIMESPAN>,<TIMESPAN>,<TIMESPAN>
```

OR with optional spaces:

```
<TIMESPAN>, <TIMESPAN>, <TIMESPAN>
```

The timespans can be absolute, relative or mixed.
Eg.: downscale over the weekend and at night:

```
Sat-Sun 00:00-24:00 Europe/Berlin, Mon-Fri 20:00-07:00 Europe/Berlin
```

### Duration

A duration can be defined either by an integer representing seconds

```text
"120" # 120 seconds (2 minutes)
"900" # 900 seconds (15 minutes)
```

Or by a duration strings:

```text
"1h30m" # 1 hour and 30 minutes
"1.5h"  # 1 ½ hours (1 hour and 30 minutes)
"2m"    # 2 minutes
"10s"   # 10 seconds
"300s"  # 300 seconds
```

Other units:

```text
"ns"      # nanoseconds
"us"/"µs" # microseconds
"ms"      # milliseconds
"s"       # seconds
"m"       # minutes
"h"       # hours
```

## Concepts

### Layers

Layers are layers of values. If the highest Layer doesn't have a value, it falls through it and tries to get it from the next lower layer.

#### Layer Hierarchy

1. [Workload Layer](#workload-layer)
2. [Namespace Layer](#namespace-layer)
3. [CLI Layer](#cli-layer)
4. [ENV Layer](#env-layer)

#### Workload Layer

Defined by the [annotations](#annotations) on the [workload](#scalable-resources).

#### Namespace Layer

Defined by the [annotations](#annotations) on the namespace.

#### CLI Layer

Defined by the [command line arguments](#arguments) at startup.

#### ENV Layer

Defined by the [environemt variables](#environment-variables) at startup.

#### Examples

> [!Note]
> A process line with "(...)" is a compacted form, instead of showing the process on each layer

```text
--- Layers
Workload: (no annotations)
Namespace: exclude=true
CLI: (defaults)
ENV: (no env vars)
--- Process:
Exclusion not specified on workload layer, going to next layer
Exclusion set to true on namespace layer, excluding workload
--- Result:
Workload is excluded, no changes will be made to it
```

```text
--- Layers
Workload: exclude=false
Namespace: exclude=true
CLI: downtime="Mon-Fri 08:00-16:00 Europe/Berlin"
ENV: (no env vars)
--- Process:
Exclusion set to false on workload layer, not excluding workload
No forced scaling found on any layer (...)
No scaling specified on Workload layer, going to next layer
No scaling specified on Namespace layer, going to next layer
Scaling "downtime" specified on CLI layer, scaling according to the downtime schedule on the cli layer
--- Result:
Workload will be scaled according to the downtime schedule on the cli layer
```

```text
--- Layers
Workload: uptime="Mon-Fri 08:00-16:00 Europe/Berlin"
Namespace: force-downtime=true
CLI: downtime="Mon-Fri 20:00-08:00 PST"
ENV: (no env vars)
--- Process:
Exclusion not set on any layer (...)
Forced scaling found on namespace layer, forcing downscale (...)
--- Result:
Workload will be forced into a down-scaled state
```

```text
--- Layers
Workload: uptime="Mon-Fri 08:00-16:00 Europe/Berlin"
Namespace: force-downtime=true
CLI: downtime="Mon-Fri 20:00-08:00 PST"
ENV: (no env vars)
--- Process:
Exclusion not set on any layer (...)
No forced scaling found on any layer (...)
Scaling "uptime" set on workload layer, scaling according to the uptime schedule on the cli layer
--- Result:
Workload will be scaled according to the uptime schedule on the cli layer
```

### Values

- <span id="downscale-period">downscale-period</span>:
  - comma seperated list of [timespans](#timespans)
  - within these periods the [workload](#scalable-resources) will be scaled down
  - incompatible with [downtime](#downtime), [uptime](#uptime)
- <span id="downtime">downtime</span>:
  - comma seperated list of [timespans](#timespans)
  - within these timespans the [workload](#scalable-resources) will be scaled down, outside of them it will be scaled up
  - incompatible with [downscale-period](#downscale-period), [upscale-period](#upscale-period), [uptime](#uptime)
- <span id="upscale-period">downscale-period</span>:
  - comma seperated list of [timespans](#timespans)
  - within these periods the [workload](#scalable-resources) will be scaled up
  - incompatible with [downtime](#downtime), [uptime](#uptime)
- <span id="uptime">uptime</span>:
  - comma seperated list of [timespans](#timespans)
  - within these timespans the [workload](#scalable-resources) will be scaled up, outside of them it will be scaled down
  - incompatible with [downscale-period](#downscale-period), [upscale-period](#upscale-period), [downtime](#downtime)
- <span id="exclude">exclude</span>:
  - boolean
  - when true, the [workload](#scalable-resources) will be excluded/ignored while scaling
- <span id="exclude-until">exclude-until</span>:
  - RFC3339 Timestamp
  - the [workload](#scalable-resources) will be excluded until this time
- <span id="force-uptime">force-uptime</span>:
  - boolean
  - if set to true the [workload](#scalable-resources) will be forced into an uptime state
  - incompatible with [force-downtime](#force-downtime)
- <span id="force-downtime">force-downtime</span>:
  - boolean
  - if set to true the [workload](#scalable-resources) will be forced into an downtime state
  - incompatible with [force-uptime](#force-uptime)
- <span id="downscale-replicas">downscale-replicas</span>:
  - int
  - the replicas that the [workload](#scalable-resources) should have while downscaled
- <span id="grace-period">grace-period</span>:
  - [duration](#duration)
  - the duration a [workload](#scalable-resources) has to exist until it is first scaled

For more info please refer to the [official documentation](https://pkg.go.dev/time#ParseDuration)

## Migrating from py-kube-downscaler

### Basic migration

<!-- TODO Basic migration -->

### Edge cases

If you had an implementation that used some of the quirks of the py-kube-downscaler you might need to change those.

Some cases where this might be needed include:

- [Incompatibility instead of priority](#diff-incompatible)
  - eg. if you had a program that dynamically added a uptime annotation on a workload with a downtime annotation because you relied on the uptime annotation taking over
- [Layer system](#diff-layer-system)
  - eg. the behaviour of excluding a namespace resulting in all workloads in it being excluded is not quite the same, as the workload could overwrite this by setting exclude to false
- [A pod that upscales the whole cluster](https://github.com/caas-team/py-kube-downscaler/blob/main/README.md?plain=1#L90)
  - this behaviour is no longer available
- [RFC3339 timestamp](#diff-uniform-timestamp)
  - if you used the short form versions of the ISO 8601 timestamp (`2023-08-12`, `2023-233` or `2023-W34-1`)
- [Actual exclusion](#diff-actual-exclusion)
  - eg. if you had a program that dynamically excluded a namespace and need it to then go in an upscaled state

### Differences to py-kube-downscaler

<span id="diff-incompatible">Incompatibility instead of priority</span>:

- some values are now incompatible instead of using one over the other
- backwards compatible: shouldn't break anything in most cases

<span id="diff-duration-units">Duration units</span>:

- instead of integers representing seconds you can also use duration strings see [Duration](#duration) for more information
- backwards compatible: fully compatible, integer seconds are still supported

<span id="diff-layer-system">Layer system</span>:

- Makes it easier and more uniform to know what configuration is going to be used. All annotations can now also be easily applied to namespaces.
- backwards compatible: shouldn't break anything in most cases

<span id="diff-explicit-include">[--explicit-include](#--explicit-include) cli argument</span>:

- a simple way to explicitly include single workloads. See [--explicit-include](#--explicit-include) for more details.
- backwards compatible: fully compatible, no prior behaviour was changed

<span id="diff-comfort-spaces">Comfort spaces</span>:

- allows for spaces in configuration to make the configuration more readable. (applies to: any comma seperated list, [absolute timespans](#configuration-of-an-absolute-timespan))
- backwards compatible: fully compatible, you can still use the configuration without spaces

<span id="diff-uniform-timestamp">Uniform timestamp</span>:

- all timestamps are RFC3339 this is more optimized for golang, more consistent and also used by kubernetes itself
- backwards compatible: mostly, unless you used a short form of ISO 8601 eg. `2023-08-12`, `2023-233` or `2023-W34-1` it should be totally fine to not change anything

<span id="diff-overlapping-days">Overlapping [relative timespans](#configuration-of-a-relative-timespan) into next day</span>:

- timespans can overlap into the "next" day (eg. `Mon-Fri 20:00-06:00 UTC`). See [Relative Timespans](#configuration-of-a-relative-timespan)
- backwards compatible: fully compatible, this didn't change any existing functionallity

<span id="diff-actual-exclusion">Actual exclusion</span>:

- [excluding a workload](#exclude) won't force the workload to be upscaled
- backwards compatible: should be fully compatible, unless your implementation relies on this

<span id="diff-iana-timezones">IANA Timezones</span>:

- the downscaler uses the [IANA timezone database](https://www.iana.org/time-zones)
- backwards compatible: fully compatible, "Olson timezones" is just a lesser known synonym for the IANA time zone database

<span id="diff-workload-errors">Workload error events</span>:

- errors with the configuration of a [workload](#scalable-resources) are shown as events on the workload
- backwards compatible: fully compatible, doesn't change any existing functionality

## Developing

Please read the [contribution manifest](./CONTRIBUTING.md)

### Cloning the Repository

```bash
git clone https://github.com/caas-team/GoKubeDownscaler.git
cd GoKubeDownscaler
```

### Setting up Pre-Commit

```bash
brew install pre-commit
pre-commit install
brew install golangci-lint
brew install gofumpt
```

### Testing the downscaler

#### running the unit tests

```bash
go test -v --cover ./...
```

#### running the downscaler locally

The downscaler can be run locally by specifying a kubeconfig to use.
The kubeconfig should have at least the permissions as the Helm Charts [role.yaml](./deployments/chart/templates/role.yaml).
The downscaler will use the current-context in the kubeconfig.

```bash
go run -k=path/to/kubeconfig # ... additional configuration
```
