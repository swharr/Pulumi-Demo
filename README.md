# Pulumi EKS Web Application

Deploys a containerized Node.js application to AWS EKS with load balancing, SSL, and DNS.

## See it in action
 To see the app working: https://pulumidemo.t8rsk8s.io

## Important Context Notes:
1. Some of this code re-uses previous infrastructure code and concepts I have used over the years. 
2. It is a pre-flight app, security was less important than usability
3. the diagram and stats for nerds page on the main app were not called out, but I was curious, so I figured this would be a good way to sandbox some questions I had
4. I chose Go, because it was the quickest, easiest to instantiate considering infrastructure was being deployed. 

## What You need.

- AWS credentials configured
- Docker running locally
- Pulumi CLI installed
- Go installed

## Get Started

```bash
pulumi up
```

Initial deployment takes approximately 25 minutes due to EKS cluster provisioning.

## Change the app value displayed.

```bash
pulumi config set app:value "your text here"
pulumi up
```

Triggers a rolling pod update with zero downtime.

## Modify application code

1. Edit `app/server.js`
2. Update image tag in `main.go` (increment version, e.g. `app-v3` to `app-v4`)
3. Run `pulumi up`

## Access cluster with kubectl

```bash
pulumi stack output kubeconfig --show-secrets > ~/.kube/eks-config.yaml
export KUBECONFIG=~/.kube/eks-config.yaml
kubectl get pods
```

## Monitor deployment

```bash
kubectl get deployments
kubectl get pods
kubectl get services
kubectl logs <pod-name>
```

## Destroy infrastructure we stood up

```bash
pulumi destroy
```

Removal takes approximately 15 minutes.

## What infrastructure are we using? 
_Designed to fit within the AWS Well Architected Framework Specs_ 

- VPC with public and private subnets across 2 availability zones
- EKS cluster with 2 t3.medium worker nodes
- ECR repository for container images
- ACM SSL certificate with DNS validation
- Route53 DNS record _(Uses an existing Test Domain)
- Network Load Balancer with HTTPS termination
- Kubernetes Deployment with 2 pod replicas
- Kubernetes Services (ClusterIP and LoadBalancer types)

## Configuration

Edit `Pulumi.devstack1.yaml`:

- `aws:region` - AWS region for deployment
- `app:value` - Text content displayed on web page

## Security functions included by default.

- Non-root container execution (UID 1001)
- Dropped Linux capabilities
- Resource limits enforced
- Private subnets for workloads
- TLS encryption in transit
