# Gateway Integration Guide

Advanced capabilities include WAF, Caching, Auth, Rate Limiting, Consistent Hashing, Health Checks, and Wasm Middleware.

Parts you will need to modify or configure:

## 1. Gateway Configuration (`config.yaml`)

The most important part of hooking up the gateway is defining your routing rules in a configuration file. Since this gateway is built on top of the Kratos gateway framework, it uses a declarative configuration structure. 

Create/modify a `config.yaml` file (usually loaded in `cmd/gateway/main.go`) to map your endpoints to your backend services and apply the middlewares.

```yaml
# Example config.yaml structure
endpoints:
  - path: /api/v1/users/*
    protocol: HTTP
    timeout: 5s
    # Apply middlewares specifically to this route
    middlewares:
      - name: auth
      - name: ratelimit
      - name: waf
      - name: hashkey
    backends:
      - target: "http://user-service-node-1:8000"
      - target: "http://user-service-node-2:8000"

  - path: /api/v1/products/*
    protocol: HTTP
    middlewares:
      - name: cache
      - name: wasm # Dynamic plugin
    backends:
      - target: "http://product-service:8080"
```

## 2. Gateway Main Entrypoint (`cmd/gateway/main.go`)

Currently, `cmd/gateway/main.go` registers all the middlewares, but you will need to ensure it is loading your specific `config.yaml` file on startup.

**What to modify:**
- Ensure the `config.Load()` function in `main.go` points to your active `config.yaml` (or accepts it via command-line flags).
- If you need to inject secrets (like the JWT signing key for the `auth` middleware or the Redis URL for the `cache` middleware), you should pass those into the middleware configurations here.

## 3. Frontend Modifications

Your frontend needs to be aware of the gateway's security and routing mechanisms.

**What to modify in your Frontend Code:**
- **Base URL:** Point all your API calls to the Gateway's exposed port (e.g., `http://localhost:8000/api/...`) instead of directly to the backend services.
- **Authentication:** Ensure the frontend attaches the proper authentication credentials (e.g., `Authorization: Bearer <token>`) so the `auth` middleware doesn't drop the requests.
- **Consistent Hashing (X-Session-ID):** For requests that must consistently hit the same backend node (like WebSockets or stateful checkout sessions), the frontend must generate and send an `X-Session-ID` header.
- **Error Handling:** 
  - Handle `429 Too Many Requests` (Rate Limiter).
  - Handle `403 Forbidden` (WAF or Wasm blocks).
  - Handle `401 Unauthorized` (Auth drops).

## 4. Backend Microservices Modifications

Your backend services sit *behind* the gateway and should trust it.

**What to modify in your Backend Services:**
- **Network Isolation:** Ensure your backend services are only accessible by the Gateway (e.g., via Docker networks, VPCs, or firewall rules) so users can't bypass the Gateway's WAF and Rate Limiter.
- **Health Checks:** The Gateway's `HealthChecker` actively pings backends using TCP (or HTTP if you expand it). Ensure your backends respond cleanly to connection probes so they aren't marked as "dead" by the gateway.
- **Header Forwarding:** If your backend needs to know the original user's IP, ensure you configure the gateway to pass `X-Forwarded-For` and that your backends read from it.

## 5. Middleware Assets

Several of the new middlewares rely on external assets or dependencies.

**What to modify:**
- **Wasm Plugins (`middleware/wasm/guest/`):** If you are using the Wasm middleware, you need to compile your actual logic into a `.wasm` file and update `wasm.go` or `config.yaml` to point to the correct file path.
- **Redis (`middleware/cache/`):** The caching middleware currently uses an in-memory map or a mocked Redis client. You will need to spin up a real Redis instance and pass the connection string to the Cache middleware.
- **WAF Rules:** If your WAF needs dynamic rules (e.g., loaded from a database or a `rules.json` file), you'll need to feed those into the WAF middleware configuration.
