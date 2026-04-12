---

### 📊 Performance Benchmarks (1,000,000 Requests)

This section documents the performance characteristics of **Nietzsche** under different architectural configurations and workloads.

---

### 🟢 Scenario 1: Single-threaded (I/O Bound)
*This is the baseline configuration where the execution logic runs on a single event loop, simulating the standard Redis threading model.*

#### 1. Benchmark Results
Tests were conducted using `redis-benchmark` with **50 parallel clients**, **3-byte payloads**, and **keep-alive** enabled.

| Command | Throughput (RPS) | Avg Latency | P50 | P95 | P99 | Max |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **SET** | **34,164.67** | 0.779 ms | 0.671 ms | 1.127 ms | 3.103 ms | 25.807 ms |
| **GET** | **34,900.36** | 0.746 ms | 0.671 ms | 1.031 ms | 1.727 ms | 10.727 ms |

#### 2. CPU Profiling & Deep Dive
To identify internal bottlenecks, I captured a CPU profile using `pprof` during the peak of the GET benchmark.

[![CPU Profile Graph](image/singleThreadIoBound.png)](image/pprof001_single_thread_io_bound.svg)
*Figure 1: Flame graph showing hotspots during the GET benchmark. Click the image to view the interactive SVG.*

**Key Insights from Profiling:**

* **System Call Dominance (~48%):** Nearly half of the CPU time is consumed by `linuxSyscall` (specifically `read` and `write`). This indicates the application is highly efficient, as the primary bottleneck is the **OS kernel's network stack**, not the user-space logic.
* **Highly Optimized RESP Parsing:** The custom RESP decoder (`core.DecodeOne`) and command dispatcher account for less than **5%** of total CPU usage. This proves the efficiency of the byte-level protocol handling.
* **Sub-millisecond Tail Latency:** The average latency remains under **0.8ms**. The stable P99 (1.7ms for GET) demonstrates consistent performance under load. Minor spikes (Max Latency 25ms) are typical for Go's Garbage Collection (GC) or OS-level context switching.

**Technical Conclusion:**
In this scenario, Nietzsche is **I/O bound**. The single-threaded execution model is sufficient to saturate the available network throughput without the overhead of lock contention or complex synchronization, making it an ideal choice for high-frequency, low-latency key-value operations.

---


### Benchmark SET performance
redis-benchmark -p 4000 -t set -n 1000000 -r 1000000

### Benchmark GET performance
redis-benchmark -p 4000 -t get -n 1000000 -r 1000000

### pprof
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30



