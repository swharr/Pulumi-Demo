package main

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/acm"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ecr"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	awsxec2 "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ec2"
	"github.com/pulumi/pulumi-docker/sdk/v4/go/docker"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	"pulumi-demoapp/pkg/webapp"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// 1) Config
		cfg := config.New(ctx, "app")
		display := cfg.Get("value")
		if display == "" {
			display = "abc123"
		}

		// 2) VPC with private and public subnets
		numAZs := 2
		strategy := awsxec2.NatGatewayStrategySingle
		vpc, err := awsxec2.NewVpc(ctx, "eks-vpc", &awsxec2.VpcArgs{
			NumberOfAvailabilityZones: &numAZs,
			NatGateways: &awsxec2.NatGatewayConfigurationArgs{
				Strategy: strategy,
			},
			SubnetSpecs: []awsxec2.SubnetSpecArgs{
				{Type: awsxec2.SubnetTypePublic},
				{Type: awsxec2.SubnetTypePrivate},
			},
		})
		if err != nil {
			return err
		}

		// 3) EKS Cluster IAM Role
		eksRole, err := iam.NewRole(ctx, "eks-cluster-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Principal": {"Service": "eks.amazonaws.com"},
					"Action": "sts:AssumeRole"
				}]
			}`),
		})
		if err != nil {
			return err
		}

		_, err = iam.NewRolePolicyAttachment(ctx, "eks-cluster-policy", &iam.RolePolicyAttachmentArgs{
			Role:      eksRole.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"),
		})
		if err != nil {
			return err
		}

		// 4) EKS Cluster
		cluster, err := eks.NewCluster(ctx, "eks-cluster", &eks.ClusterArgs{
			RoleArn: eksRole.Arn,
			VpcConfig: &eks.ClusterVpcConfigArgs{
				SubnetIds: pulumi.StringArrayOutput(pulumi.All(vpc.PublicSubnetIds, vpc.PrivateSubnetIds).ApplyT(
					func(args []interface{}) []string {
						public := args[0].([]string)
						private := args[1].([]string)
						return append(public, private...)
					}).(pulumi.StringArrayOutput)),
			},
		})
		if err != nil {
			return err
		}

		// 5) Node Group IAM Role
		nodeRole, err := iam.NewRole(ctx, "eks-node-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Principal": {"Service": "ec2.amazonaws.com"},
					"Action": "sts:AssumeRole"
				}]
			}`),
		})
		if err != nil {
			return err
		}

		policies := []string{
			"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
			"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
			"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
		}
		for i, policy := range policies {
			_, err = iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("node-policy-%d", i), &iam.RolePolicyAttachmentArgs{
				Role:      nodeRole.Name,
				PolicyArn: pulumi.String(policy),
			})
			if err != nil {
				return err
			}
		}

		// 6) EKS Managed Node Group
		_, err = eks.NewNodeGroup(ctx, "eks-node-group", &eks.NodeGroupArgs{
			ClusterName:   cluster.Name,
			NodeRoleArn:   nodeRole.Arn,
			SubnetIds:     vpc.PrivateSubnetIds,
			InstanceTypes: pulumi.StringArray{pulumi.String("t3.medium")},
			ScalingConfig: &eks.NodeGroupScalingConfigArgs{
				DesiredSize: pulumi.Int(2),
				MinSize:     pulumi.Int(2),
				MaxSize:     pulumi.Int(2),
			},
		}, pulumi.DependsOn([]pulumi.Resource{eksRole}))
		if err != nil {
			return err
		}

		// 7) ECR repo for your image
		repo, err := ecr.NewRepository(ctx, "app-ecr", &ecr.RepositoryArgs{
			ForceDelete: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		// Get AWS account and region info
		reg, err := aws.GetCallerIdentity(ctx, nil, nil)
		if err != nil {
			return err
		}
		region, err := aws.GetRegion(ctx, nil, nil)
		if err != nil {
			return err
		}
		imageName := pulumi.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:app-v3",
			reg.AccountId, region.Name, repo.Name)

		// 8) Get ECR authorization credentials
		authPassword := pulumi.All(repo.RegistryId).ApplyT(func(args []interface{}) (string, error) {
			registryId := args[0].(string)
			creds, err := ecr.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenArgs{
				RegistryId: &registryId,
			})
			if err != nil {
				return "", err
			}
			return creds.Password, nil
		}).(pulumi.StringOutput)

		server := pulumi.Sprintf("%s.dkr.ecr.%s.amazonaws.com", reg.AccountId, region.Name)

		// 9) Build & push the image to ECR
		_, err = docker.NewImage(ctx, "app-image", &docker.ImageArgs{
			ImageName: imageName,
			Build: &docker.DockerBuildArgs{
				Context:  pulumi.String("./app"),
				Platform: pulumi.StringPtr("linux/amd64"),
			},
			Registry: &docker.RegistryArgs{
				Server:   server,
				Username: pulumi.String("AWS"),
				Password: authPassword,
			},
		}, pulumi.DependsOn([]pulumi.Resource{repo}))
		if err != nil {
			return err
		}

		// 10) Get Route53 hosted zone
		zone, err := route53.LookupZone(ctx, &route53.LookupZoneArgs{
			Name: pulumi.StringRef("t8rsk8s.io"),
		})
		if err != nil {
			return err
		}

		// 11) Create ACM certificate
		domainName := "pulumidemo.t8rsk8s.io"
		cert, err := acm.NewCertificate(ctx, "cert", &acm.CertificateArgs{
			DomainName:       pulumi.String(domainName),
			ValidationMethod: pulumi.String("DNS"),
		})
		if err != nil {
			return err
		}

		// 12) Create Route53 record for DNS validation
		certValidationRecord, err := route53.NewRecord(ctx, "cert-validation", &route53.RecordArgs{
			Name: cert.DomainValidationOptions.Index(pulumi.Int(0)).ResourceRecordName().Elem(),
			Type: cert.DomainValidationOptions.Index(pulumi.Int(0)).ResourceRecordType().Elem(),
			Records: pulumi.StringArray{
				cert.DomainValidationOptions.Index(pulumi.Int(0)).ResourceRecordValue().Elem(),
			},
			ZoneId: pulumi.String(zone.ZoneId),
			Ttl:    pulumi.Int(60),
		})
		if err != nil {
			return err
		}

		// 13) Wait for certificate validation
		certValidation, err := acm.NewCertificateValidation(ctx, "cert-validation-waiter", &acm.CertificateValidationArgs{
			CertificateArn: cert.Arn,
			ValidationRecordFqdns: pulumi.StringArray{
				certValidationRecord.Fqdn,
			},
		})
		if err != nil {
			return err
		}

		// 14) Generate kubeconfig
		kubeconfig := pulumi.All(cluster.Endpoint, cluster.CertificateAuthority, cluster.Name).ApplyT(
			func(args []interface{}) string {
				endpoint := args[0].(string)
				certData := args[1].(eks.ClusterCertificateAuthority).Data
				clusterName := args[2].(string)
				kubeConfig := fmt.Sprintf(`apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: %s
    server: %s
  name: %s
contexts:
- context:
    cluster: %s
    user: %s
  name: %s
current-context: %s
kind: Config
preferences: {}
users:
- name: %s
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: aws
      args:
        - eks
        - get-token
        - --cluster-name
        - %s
`, *certData, endpoint, clusterName, clusterName, clusterName, clusterName, clusterName, clusterName, clusterName); return kubeConfig
			}).(pulumi.StringOutput)

		// 15) K8s provider scoped to this cluster
		k8sProvider, err := kubernetes.NewProvider(ctx, "k8s", &kubernetes.ProviderArgs{
			Kubeconfig: kubeconfig,
		}, pulumi.DependsOn([]pulumi.Resource{cluster}))
		if err != nil {
			return err
		}

		// 16) Deploy the app via ComponentResource
		wa, err := webapp.NewWebApp(ctx, "hello", &webapp.WebAppArgs{
			Image:        imageName,
			DisplayValue: pulumi.String(display),
			Replicas:     pulumi.IntPtr(2),
			Namespace:    pulumi.StringPtr("default"),
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}

		// 17) Create LoadBalancer Service with HTTPS
		appLabels := pulumi.StringMap{
			"app": pulumi.String("hello"),
		}

		lbService, err := corev1.NewService(ctx, "hello-lb", &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("hello-lb"),
				Namespace: pulumi.String("default"),
				Annotations: pulumi.StringMap{
					"service.beta.kubernetes.io/aws-load-balancer-type":             pulumi.String("nlb"),
					"service.beta.kubernetes.io/aws-load-balancer-ssl-cert":         certValidation.CertificateArn,
					"service.beta.kubernetes.io/aws-load-balancer-backend-protocol": pulumi.String("http"),
					"service.beta.kubernetes.io/aws-load-balancer-ssl-ports":        pulumi.String("443"),
				},
			},
			Spec: &corev1.ServiceSpecArgs{
				Type:     pulumi.String("LoadBalancer"),
				Selector: appLabels,
				Ports: corev1.ServicePortArray{
					&corev1.ServicePortArgs{
						Name:       pulumi.String("https"),
						Port:       pulumi.Int(443),
						TargetPort: pulumi.Int(3000),
						Protocol:   pulumi.String("TCP"),
					},
					&corev1.ServicePortArgs{
						Name:       pulumi.String("http"),
						Port:       pulumi.Int(80),
						TargetPort: pulumi.Int(3000),
						Protocol:   pulumi.String("TCP"),
					},
				},
			},
		}, pulumi.Provider(k8sProvider), pulumi.DependsOn([]pulumi.Resource{wa.Deployment}))
		if err != nil {
			return err
		}

		// 18) Create Route53 DNS record
		lbHostname := lbService.Status.LoadBalancer().Ingress().Index(pulumi.Int(0)).Hostname()

		_, err = route53.NewRecord(ctx, "app-dns", &route53.RecordArgs{
			Name:   pulumi.String(domainName),
			Type:   pulumi.String("CNAME"),
			ZoneId: pulumi.String(zone.ZoneId),
			Records: pulumi.StringArray{
				lbHostname.Elem(),
			},
			Ttl: pulumi.Int(300),
		})
		if err != nil {
			return err
		}

		// 19) Exports
		ctx.Export("region", pulumi.String(region.Name))
		ctx.Export("clusterName", cluster.Name)
		ctx.Export("kubeconfig", kubeconfig)
		ctx.Export("vpcId", vpc.VpcId)
		ctx.Export("serviceName", wa.Service.Metadata.Name())
		ctx.Export("url", pulumi.Sprintf("https://%s", domainName))
		ctx.Export("certificateArn", cert.Arn)
		ctx.Export("loadBalancerHostname", lbHostname)

		return nil
	})
}
