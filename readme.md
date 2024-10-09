# Distributed Cache System

This project implements a distributed caching solution inspired by common practices in distributed systems, such as consistent hashing, LRU caching, and single-flight request deduplication. The primary goal is to provide a scalable and efficient caching mechanism that can handle high request rates, prevent cache breakdowns, and distribute data evenly across nodes.

## Features

- **LRU (Least Recently Used) Cache**: Each node in the system maintains an LRU cache to manage cached data, ensuring that the most frequently accessed data is readily available and less-used data is evicted as needed.
  
- **Consistent Hashing**: The system employs consistent hashing to distribute cached data across multiple nodes. This helps minimize data movement when nodes are added or removed, ensuring balanced load distribution.

- **SingleFlight for Request Deduplication**: To prevent cache breakdowns, the system leverages a "single-flight" mechanism, where duplicate requests for the same key are merged into a single request. This prevents redundant calls to the backend and reduces the load on data sources.

- **HTTP Communication Between Nodes**: Nodes communicate with each other via HTTP. Each node can query others for data, ensuring that data can be accessed seamlessly even if it resides on a different server.

## Project Structure

- `lru/`: Implements an LRU cache to manage data locally within each cache node.
- `consistenthash/`: Contains the consistent hashing implementation, responsible for evenly distributing keys among nodes.
- `singleflight/`: Provides request deduplication to prevent multiple redundant loads for the same key.
- `distributecache/`: Manages core caching functionality, including peer-to-peer communication and handling HTTP requests.

## Interfaces for Remote Access

- **PeerPicker** is used to pick a peer for a given key.
- **PeerGetter** is used to get data from a remote peer.

## How It Works

### Data Request Flow

1. A client requests data for a given key.
2. The Group checks if the key is available in the local LRU cache.
3. If the data is not found locally, the `HttpPool` (implementing `PeerPicker`) is used to select a remote peer via consistent hashing.
4. If a suitable peer is found, an HTTP request is made to fetch the data from that peer.
5. If no peer has the data, the system falls back to a data source using the `Getter` function, and the fetched data is added to the cache.

### SingleFlight Prevention

To avoid multiple cache misses for the same key causing a load spike, `singleflight.Group` ensures only one request to the backend is made for each key at a time.

### Consistent Hashing

Consistent hashing distributes keys evenly across nodes, and helps maintain balanced load distribution when nodes join or leave the system, thereby reducing the number of keys that need to be remapped.

## Future Improvements

To make this distributed caching system more production-ready and robust, several areas can be optimized:

1. **Service Discovery and Coordination with etcd/Consul**  
   Currently, nodes are manually configured, and there is no dynamic mechanism for service discovery. Integrating etcd or Consul would allow nodes to automatically discover peers, making the system more fault-tolerant and easy to scale. Service discovery tools could help manage node registration, track node availability, and automatically adjust the consistent hashing ring when nodes are added or removed.

2. **Use RPC for Inter-Node Communication**  
   The system currently uses HTTP for communication between nodes, which introduces additional overhead in terms of latency and serialization/deserialization of data. Replacing HTTP with a more efficient RPC (Remote Procedure Call) mechanism, such as gRPC, would provide lower latency, better performance, and strong data typing. This could improve the overall efficiency of inter-node communication, especially in high-throughput environments.

3. **Adding a Distributed Lock Mechanism**  
   The cache currently uses a simple mutex for managing access to shared resources. Introducing a distributed locking mechanism, such as etcd's lease or a Redis-based lock, would make the system more robust in scenarios where multiple nodes could attempt to update the same resource concurrently, particularly in cases where nodes share responsibilities.

4. **Advanced Consistency Mechanism**  
   Implementing cache consistency mechanisms to keep data up-to-date across distributed nodes would improve reliability. Strategies such as write-through, write-behind, or invalidations could be implemented to ensure the data in different caches remains consistent. Depending on the use case, this could be paired with eventual or strong consistency guarantees.

5. **Monitoring and Metrics Collection**  
   Adding monitoring and metrics via Prometheus and Grafana would allow for performance tracking and system health monitoring. Observability is crucial in distributed systems, as it helps identify bottlenecks, node failures, and potential inconsistencies.

## How to Run

To set up the distributed cache system:

1. Clone the repository.
2. Start multiple instances of the distributed cache node (`HttpPool`) with different addresses.
3. Configure the nodes to be aware of each other using the `Set()` method, or use etcd/Consul for automatic service discovery.
4. Use a client to interact with the nodes by querying data using the HTTP endpoints provided by each node.

## Acknowledgments

This project was inspired by concepts from groupcache and articles from 极客兔兔. We appreciate their valuable insights and contributions.
