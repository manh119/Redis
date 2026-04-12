# A high-performance, distributed in-memory key-value store built in Go


![img_2.png](img_2.png)


## 🏗️ Architecture

![img.png](docs/img.png)

Detailed explanation with diagrams is available in [docs/architecture.md](docs/architecture.md).

## ⚙️ Features

- **RESP Protocol**  
  Full support for Redis Serialization Protocol.

- **Data Structures**
    - Hash dictionary
    - SkipList (for sorted sets)
    - Bloom Filter
    - Count-Min Sketch
    - Eviction Pool

- **Concurrency Models**
    - Single-threaded IO-bound
    - Epoll-based IO multiplexing
    - Thread-per-connection
    - Thread pool architecture

- **Graceful Shutdown**  
  - With resource cleanup.

- **Benchmarking & Profiling**
    - CPU-bound vs IO-bound workloads
    - Multi-thread vs single-thread performance
    - Visualizations with `pprof`


## 📌 Supported Commands

| Data Structure            | Commands |
|---------------------------|----------|
| **Key-Value Store**       | `PING`, `GET`, `SET`, `TTL`, `EXPIRE`, `DEL`, `EXISTS`, `FLUSHDB`, `PERSIST` |
| **Set**                   | `SADD`, `SISMEMBER`, `SREM`, `SMEMBERS` |
| **Sorted Set (SkipList)** | `ZADD`, `ZSCORE`, `ZRANK` |
| **Count-Min Sketch (CMS)**| `CMS.INITBYPROB`, `CMS.INCRBY`, `CMS.QUERY` |
| **Bloom Filter (BF)**     | `BF.RESERVE`, `BF.MADD`, `BF.MEXISTS` |

## 📊 Benchmarks with 1M request

**Highlights:**
- **Single-thread IO-bound**: Efficient for lightweight workloads.
- **Thread pool architecture**: Scales better under heavy concurrent connections.
- **Bloom Filter & CMS**: Sub-linear memory usage with probabilistic guarantees.

![img_1.png](docs/benchmark.png)

Full benchmark results and profiling visualizations are in [docs/benchmark.md](docs/benchmark.md).

## ▶️ Quick Start within 10s

```bash
git clone https://github.com/manh119/Redis.git
cd Redis
go run cmd/main.go

# Connect to server using Redis CLI
redis-cli -p 4000
```

## 🚀 Future Enhancements
- [ ] Persistent storage (file-based or database)
- [ ] Pub/Sub messaging
- [ ] Replication
- [ ] Cluster support
