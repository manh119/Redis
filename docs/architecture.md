
# Architecture

---

## Thread-Per-Connection Model
![img_1.png](image/ThreadPerConnection.png)
- each request have a thread to handle
---
## Thread-Pool Model
![img.png](image/threadPoolArchitect.png)
- pool of fix-size thread, no need to create/destroy each time need or no need in thread per connection model
---
## Multiplexing model
![img_2.png](image/EpollMonitor.png)
- epoll to monitor tcp listener (for new connection) and connection (for new command)
- event loop 
---
## Multi-thread with shared nothing architecture
![img_6.png](image/sharedNothingArchitecture.png)
- each worker run in a go routine and own its dictionary (hashmap)
- IO handler forward to correct worker by workerId = hash(key)
---
## Skip list for sorted set
![img_4.png](image/SkipList.png)
- improve search in linked list O(n) -> O(logn) by skip node
---
## Count min sketch for 100M element ~ 8MB
![img_5.png](image/countMinSketch.png)
- Trade off beetween Accuracy - Size memory 