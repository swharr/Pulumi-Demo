# EKS Demo App

Quick demo showing how to deploy a containerized app to EKS with Pulumi. Stood this up for a take-home exercise.

## What it does

Spins up:
- EKS cluster (2 nodes, t3.medium)
- VPC with public/private subnets
- ECR repo + builds/pushes container
- NLB with SSL termination
- Route53 DNS
- All the usual IAM stuff

The app itself is just a simple Express server that displays a configurable value and some stats about the deployment.

## Prerequisites

- AWS CLI configured
- Docker running
- Pulumi CLI
- Go 1.18+
- A Route53 hosted zone (currently hardcoded to `t8rsk8s.io` - this is my domain - you'll want to change it)

## Deploy

```bash
pulumi up
```

Takes about 20-25 mins because EKS is slow.

## Configuration

Set a custom display value:
```bash
pulumi config set websrv1:setting "whatever you want"
pulumi up
```

## Update the app

1. Edit `app/server.js`
2. Bump version in `main.go` (line 150-ish, change `app-v4` to `app-v5`)
3. `pulumi up`

## Access the cluster

```bash
pulumi stack output kubeconfig --show-secrets > ~/.kube/config
kubectl get pods
```

## Tear down

```bash
pulumi destroy
```

## Notes

- Using single NAT gateway to save costs (not prod-ready)
- Security is pretty locked down (non-root containers, capability dropping, etc.)
- The webapp component in `pkg/webapp/` makes it reusable
- Health checks on `/healthz` and `/readyz`
- Graceful shutdown handles SIGTERM properly

## TODO

- [ ] Make domain configurable instead of hardcoded
- [ ] Add monitoring/observability
- [ ] Maybe switch to Fargate to avoid managing nodes?
