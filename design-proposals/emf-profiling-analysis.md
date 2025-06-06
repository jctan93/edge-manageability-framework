# EMF profiling: Analyze the timing and resource consumption of EMF with various profiles

Author(s): Krishna

Last updated: 05.06.2025

## Abstract

The objective of this exercise is to capture the timing and resource consumption of EMF when using different profiles. This will help in understanding the timing and resource consumption and guide future optimizations. For this exercise, we will use the EMF 3.0 release and coder instance.

## Tools used

- `kubectl top node` : Shows current real-time usage of CPU and memory.Data is fetched from the Metrics Server, which collects usage stats from kubelet.
- `kubectl describe node` : Shows the requested and allocatable resources, and also total capacity. This includes the sum of CPU and memory requests and limits from all pods on the node.
- Linux commands (instantaneous resource usage):
  - CPU: `top -bn1 | grep "Cpu(s)" | awk '{print "CPU Used: " $2 + $4 "%, CPU Free: " $8 "%"}'`
  - Memory: `free | awk '/Mem:/ {used=$3; total=$2; printf "Memory Used: %.2f%%, Memory Free: %.2f%%\n", used/total*100, $4/total*100}'`
  - Storage: `df --total -k | awk '/^total/ {used=$3; total=$2; free=$4; printf "Disk Used: %.2f%%, Disk Free: %.2f%%\n", used/total*100, free/total*100}'`
- instrumentation in the EMF code base

## Codebase and configuration

3.0 tag of EMF codebase is used for this exercise. EMF supports pre-defined deployment presets, which can be used to deploy the EMF instance with different profiles. The following profiles are used for this exercise: `dev-internal-coder-autocert.yaml` as baseline. Changes made to the presets will be captured as part of the profiling data.

## Profiling data

Two coder instance are deployed with same overall resource configuration

- CPU: 16 cores
- Memory: 64 GiB
- Storage: 140 GiB

### Comparing `dev-internal-coder-autocert.yaml` with `enableObservability` set to `false`

#### Kubernetes Resource allocation

Baseline profile `dev-internal-coder-autocert.yaml` with `enableObservability` set to `true`:

```sh
  Resource           Requests     Limits
  --------           --------     ------
  cpu                2466m (15%)  14957550m (93484%)
  memory             4603Mi (7%)  15308700Mi (24200%)  
```

```sh
NAME                 CPU(cores)   CPU%   MEMORY(bytes)   MEMORY%
kind-control-plane   4164m        26%    39464Mi         62%
```

Baseline profile `dev-internal-coder-autocert.yaml` with `enableObservability` set to `false`:

```sh
  Resource           Requests     Limits
  --------           --------     ------
  cpu                1823m (11%)  8646800m (54042%)
  memory             2169Mi (3%)  8850844Mi (13991%)
```

```sh
NAME                 CPU(cores)   CPU%   MEMORY(bytes)   MEMORY%
kind-control-plane   1507m        9%     19023Mi         30%
```

#### Linux Resource allocation

Baseline profile `dev-internal-coder-autocert.yaml` with `enableObservability` set to `true`:

```sh
CPU Used: 22.6%, CPU Free: 76.3%
Memory Used: 54.66%, Memory Free: 17.65%
Disk Used: 33.24%, Disk Free: 64.32%
```

Baseline profile `dev-internal-coder-autocert.yaml` with `enableObservability` set to `false`:

```sh
CPU Used: 6%, CPU Free: 94.0%
Memory Used: 24.48%, Memory Free: 62.97%
Disk Used: 22.89%, Disk Free: 74.31%
```

### Takeaways

- enabling observability in EMF significantly increases the resource consumption, especially in terms of CPU and memory.
