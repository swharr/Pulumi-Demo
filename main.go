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
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"pulumi-demoapp/pkg/webapp"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "websrv1")
		display := cfg.Get("setting")
		if display == "" {
			display = "abc123" // default fallback
		}

		// Setup VPC - 2 AZs, single NAT for cost savings
		numAZs := 2
		vpc, err := awsxec2.NewVpc(ctx, "eks-vpc", &awsxec2.VpcArgs{
			NumberOfAvailabilityZones: &numAZs,
			NatGateways: &awsxec2.NatGatewayConfigurationArgs{
				Strategy: awsxec2.NatGatewayStrategySingle,
			},
			SubnetSpecs: []awsxec2.SubnetSpecArgs{
				{Type: awsxec2.SubnetTypePublic},
				{Type: awsxec2.SubnetTypePrivate},
			},
		})
		if err != nil {
			return err
		}

		// EKS cluster role
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

		// Create the cluster - this takes forever
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

		// Node group role + policies
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

		// Attach the standard EKS node policies
		policies := []string{
			"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
			"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
			"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
		}
		for i, policyArn := range policies {
			_, err = iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("node-policy-%d", i), &iam.RolePolicyAttachmentArgs{
				Role:      nodeRole.Name,
				PolicyArn: pulumi.String(policyArn),
			})
			if err != nil {
				return err
			}
		}

		// Spin up node group - t3.medium should be plenty
		nodeGroup, err := eks.NewNodeGroup(ctx, "eks-node-group", &eks.NodeGroupArgs{
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

		// ECR repo for container images
		repo, err := ecr.NewRepository(ctx, "app-ecr", &ecr.RepositoryArgs{
			ForceDelete: pulumi.Bool(true), // easier cleanup for demos
		})
		if err != nil {
			return err
		}

		reg, err := aws.GetCallerIdentity(ctx, nil, nil)
		if err != nil {
			return err
		}
		region, err := aws.GetRegion(ctx, nil, nil)
		if err != nil {
			return err
		}

		imageName := pulumi.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:app-v4",
			reg.AccountId, region.Name, repo.Name)

		// Get ECR creds for docker push
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

		// Build and push container
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

		// Grab existing Route53 zone
		zone, err := route53.LookupZone(ctx, &route53.LookupZoneArgs{
			Name: pulumi.StringRef("t8rsk8s.io"),
		})
		if err != nil {
			return err
		}

		// SSL cert via ACM
		domainName := "pulumidemo.t8rsk8s.io"
		cert, err := acm.NewCertificate(ctx, "cert", &acm.CertificateArgs{
			DomainName:       pulumi.String(domainName),
			ValidationMethod: pulumi.String("DNS"),
		})
		if err != nil {
			return err
		}

		// DNS validation record for cert
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

		// Wait for cert validation to complete
		certValidation, err := acm.NewCertificateValidation(ctx, "cert-validation-waiter", &acm.CertificateValidationArgs{
			CertificateArn: cert.Arn,
			ValidationRecordFqdns: pulumi.StringArray{
				certValidationRecord.Fqdn,
			},
		})
		if err != nil {
			return err
		}

		// Generate kubeconfig - probably a better way to do this but works
		kubeconfig := pulumi.All(cluster.Endpoint, cluster.CertificateAuthority, cluster.Name).ApplyT(
			func(args []interface{}) string {
				endpoint := args[0].(string)
				certData := args[1].(eks.ClusterCertificateAuthority).Data
				clusterName := args[2].(string)
				return fmt.Sprintf(`apiVersion: v1
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
`, *certData, endpoint, clusterName, clusterName, clusterName, clusterName, clusterName, clusterName, clusterName)
			}).(pulumi.StringOutput)

		// K8s provider for the cluster
		k8sProvider, err := kubernetes.NewProvider(ctx, "k8s", &kubernetes.ProviderArgs{
			Kubeconfig: kubeconfig,
		}, pulumi.DependsOn([]pulumi.Resource{cluster}))
		if err != nil {
			return err
		}

		// Pull instance type from node group for metadata
		instanceType := nodeGroup.InstanceTypes.ApplyT(func(types []string) string {
			if len(types) > 0 {
				return types[0]
			}
			return "unknown"
		}).(pulumi.StringOutput)

		tlsStatus := cert.Arn.ApplyT(func(arn string) string {
			if arn != "" {
				return "enabled (ACM)"
			}
			return "disabled"
		}).(pulumi.StringOutput)

		// Deploy app using component resource
		wa, err := webapp.NewWebApp(ctx, "hello", &webapp.WebAppArgs{
			Image:        imageName,
			DisplayValue: pulumi.String(display),
			Replicas:     pulumi.IntPtr(2),
			Namespace:    pulumi.StringPtr("default"),
			Region:       pulumi.String(region.Name),
			InstanceType: instanceType,
			ServiceType:  pulumi.String("LoadBalancer"),
			DNS:          pulumi.String(domainName),
			TLS:          tlsStatus,
			NLB:          pulumi.String("enabled"),
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}

		// LoadBalancer service with NLB + SSL
		appLabels := pulumi.StringMap{"app": pulumi.String("hello")}

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

		// Point DNS at the LB
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

		// Export useful info
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
