# Hosting the Gateway

Since the gateway is built in Go, it compiles down to a **single, highly-performant static binary**. This makes it incredibly easy and lightweight to host compared to Node.js or Python applications.

Here are the most common and effective ways to host your gateway, depending on your infrastructure:

## 1. Containerized (Docker / Kubernetes) - *Recommended*

Because the gateway sits in front of your microservices, containerizing it is usually the best approach. 

- **How it works:** You write a simple `Dockerfile` to compile the Go code into a scratch (empty) or alpine container. 
- **Where to host:** 
  - **Kubernetes (K8s):** If your backends are in K8s, deploy the gateway as an Ingress Controller or a front-facing deployment service.
  - **Managed Containers:** AWS ECS/Fargate, Google Cloud Run, or Azure Container Apps.
- **Pros:** Highly scalable, easy to integrate into your CI/CD pipelines, and keeps the gateway in the same network as your backend services.

## 2. Virtual Machines (IaaS)

If you prefer managing raw servers, you can run the compiled binary directly on a Virtual Machine.

- **How it works:** You run `go build -o gateway ./cmd/gateway`, secure copy (`scp`) the binary and your `config.yaml` to your server, and set it up as a background service using `systemd`.
- **Where to host:** AWS EC2, Google Compute Engine, DigitalOcean Droplets, or Linode.
- **Pros:** Maximum control over the network, extremely cheap for high throughput (no container overhead), and easy to set up a static IP.

## 3. Platform as a Service (PaaS)

If you want to completely avoid managing infrastructure and just want it live instantly.

- **How it works:** You push your GitHub repository directly to the PaaS provider. They automatically detect it's a Go application, compile it, and host it.
- **Where to host:** [Fly.io](https://fly.io), [Render](https://render.com), or Heroku.
- **Pros:** Zero DevOps required. Fly.io in particular is excellent for Go gateways because they deploy your app at the edge (close to users globally), reducing latency.

## Managing Middleware Assets

Keep in mind that some of the middlewares we added have infrastructure requirements:
* **Caching (`middleware/cache/`):** You will need to host a Redis instance (e.g., AWS ElastiCache or Redis Labs) and provide the connection URL to the gateway.
* **WebAssembly (`middleware/wasm/`):** You need a way to distribute your `.wasm` plugin files to the gateway (either baked into the Docker image, downloaded from an S3 bucket on startup, or mounted via a persistent volume).
