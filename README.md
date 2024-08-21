# GoKubeDownscaler

<a href="/../../releases/" title="GitHub Release" ><img alt="GitHub Release" src="https://img.shields.io/github/v/release/caas-team/GoKubeDownscaler?style=flat"></a>
<a href="./LICENSE" title="GitHub License"><img alt="GitHub License" src="https://img.shields.io/github/license/caas-team/GoKubeDownscaler?style=flat"></a>
<a href="/../../graphs/contributors" title="Contributers" ><img alt="Contributers" src="https://img.shields.io/github/contributors/caas-team/GoKubeDownscaler?style=flat"></a>
<a href="/../../stargazers" title="Stars"><img alt="Stars" src="https://img.shields.io/github/stars/caas-team/GoKubeDownscaler?style=flat"></a>

A vertical autoscaler for Kubernetes workloads.
This is a golang port of the popular [(py-)kube-downscaler](github.com/caas-team/py-kube-downscaler) with improvements and quality of life changes.

<!-- Don't use any heading smaller than h4(####) -->

## Table of contents

<!-- Keep this updated. Do not add h4(####) or smaller in this list -->

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
- [Differences to py-kube-downscaler](#differences-to-py-kube-downscaler)
- [Developing](#developing)
  - [Cloning the Repository](#cloning-the-repository)
  - [Setting up Pre-Commit](#setting-up-pre-commit)
  - [Testing the downscaler](#testing-the-downscaler)

## Scalable Resources

These are the resources the Downscaler can scale:

<!-- Keep this list updated -->

- <span id="deployments">Deployments</span>:
  - sets the replica count to the downscale replicas

## Installation

Installation is done via the [Helm Chart](./deployments/chart/README.md)

## Configuration

### Annotations

<!-- TODO Annotations -->

### Arguments

<!-- TODO Arguments -->

### Environment Variables

<!-- TODO Environment Variables -->

### Timespans

There are two different kinds of Timespans.

- Absolute Timespans: a timespan defined by two RFC3339 timestamps
- Relative Timespans: reoccuring on a schedule

#### Configuration of an Absolute Timespan:

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
Exclution not specified on workload layer, going to next layer
Exclution set to true on namespace layer, excluding workload
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
Exclution set to false on workload layer, not excluding workload
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
Exclution not set on any layer (...)
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
Exclution not set on any layer (...)
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
- <span id="uptime">downtime</span>:
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
- <span id="time-annotation">time-annotation</span>:
  - string key of an annotation with an RFC3339 Timestamp
  - when set grace-period will use the timestamp in the annotation instead of the creation time of the workload

For more info please refer to the [official documentation](https://pkg.go.dev/time#ParseDuration)

## Migrating from py-kube-downscaler

<!-- TODO Migrating from py-kube-downscaler -->

## Differences to py-kube-downscaler

<!-- TODO Differences to py-kube-downscaler -->

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
