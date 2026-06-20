# Gateway Features Overview

This document outlines all the advanced features recently implemented in the Kratos Gateway, explaining how they work and their architectural role.

## 1. Web Application Firewall (WAF) Middleware
**Location:** `middleware/waf/`

The WAF middleware provides an essential security perimeter for the gateway. It acts as the first line of defense against malicious requests. 
- **Mechanism:** Inspects HTTP incoming request paths (and potentially headers/bodies) against a predefined set of blocked patterns or rules.
- **Action:** If a request matches a malicious pattern (e.g., `/admin`, `/internal`), it immediately terminates the request and returns a `403 Forbidden` status before the request ever reaches the backend services.

## 2. In-Memory Caching Middleware
**Location:** `middleware/cache/`

The caching middleware optimizes read-heavy endpoints by serving responses directly from the gateway layer, significantly reducing backend load and latency.
- **Mechanism:** Intercepts `GET` requests and checks a distributed cache (e.g., Redis). If a valid response is found (Cache Hit), it is immediately returned to the client.
- **Action:** If a response is not found (Cache Miss), the request is forwarded to the backend, and the successful response is intercepted on its way out, stored in the cache with a defined TTL, and then sent to the client.

## 3. JWT Auth Middleware
**Location:** `middleware/auth/`

The Auth middleware centralizes authentication, ensuring only verified users can access protected backend routes.
- **Mechanism:** Reads the `Authorization` HTTP header from incoming requests. It extracts the Bearer token and verifies its cryptographic signature using a shared secret or public key.
- **Action:** If the token is valid, the request proceeds. If the token is missing, expired, or invalid, it returns a `401 Unauthorized` status.

## 4. Active Health Checks
**Location:** `client/healthcheck.go`

Instead of discovering that a backend node is dead when a user request fails (Passive Health Checks), the Active Health Check system runs constantly in the background to proactively monitor nodes.
- **Mechanism:** A background goroutine periodically (e.g., every 5 seconds) pings all registered backend nodes using a TCP connection probe.
- **Action:** If a node fails the ping (connection refused, timeout), its status is marked as `false` in the gateway registry. The load balancing picker is immediately notified to exclude this node from future request routing, ensuring zero downtime.

## 5. Token-Bucket Rate Limiting Middleware
**Location:** `middleware/ratelimit/`

Rate limiting protects the backend services from being overwhelmed by too many requests (e.g., DDoS attacks, scrapers, or misconfigured clients).
- **Mechanism:** Utilizes the Token Bucket algorithm (via `golang.org/x/time/rate`). A bucket is assigned a specific capacity (burst) and refilled at a constant rate.
- **Action:** When a request comes in, it attempts to consume a token. If tokens are available, the request proceeds. If the bucket is empty, the gateway immediately rejects the request with a `429 Too Many Requests` status.

## 6. Consistent Hashing & Hashkey Middleware
**Location:** `client/consistenthash/` and `middleware/hashkey/`

Consistent Hashing is an advanced load-balancing algorithm designed to ensure that the same client is consistently routed to the same backend node, which is critical for stateful workloads like WebSockets or in-memory session states.
- **Mechanism:** 
  1. The `Hashkey` middleware extracts a unique identifier from the incoming request. In this gateway, it extracts the `X-Session-ID` header and injects it into the request Context.
  2. The `ConsistentHash` load balancer selector uses a hash ring with virtual nodes. It reads the injected key from the Context and hashes it to find the nearest point on the ring, determining the backend node.
- **Action:** Traffic is evenly distributed, but identical `X-Session-ID` values will always hit the same exact node unless that node goes down (in which case traffic gracefully shifts to the next nearest node).

## 7. WebAssembly (Wasm) Middleware Plugins
**Location:** `middleware/wasm/`

Wasm support represents the ultimate extension mechanism for the gateway. It allows developers to write custom gateway logic in almost any language (Go, Rust, C++), compile it to a `.wasm` binary, and hot-load it into the gateway.
- **Mechanism:** Uses the `wazero` pure-Go Wasm runtime. The gateway creates an isolated, highly-secure Wasm execution sandbox for every single HTTP request.
- **Host ABI:** The gateway exposes specialized host functions to the Wasm guest:
  - `get_uri`: Copies the request URI directly into Wasm memory.
  - `block_request`: Allows the Wasm plugin to instantly flag a request to be blocked.
- **Action:** The Wasm plugin is executed dynamically. Depending on its output, the gateway can seamlessly forward the request or block it (returning a `403 Forbidden`).
