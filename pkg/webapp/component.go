package webapp

import (
	"fmt"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewWebApp(ctx *pulumi.Context, name string, args *WebAppArgs, opts ...pulumi.ResourceOption) (*WebApp, error) {
	if args == nil {
		args = &WebAppArgs{}
	}

	component := &WebApp{}
	if err := ctx.RegisterComponentResource("examples:webapp:WebApp", name, component, opts...); err != nil {
		return nil, err
	}

	ns := args.Namespace
	if ns == nil {
		ns = pulumi.StringPtr("default")
	}

	replicas := pulumi.IntPtr(1)
	if args.Replicas != nil {
		replicas = args.Replicas
	}

	labels := pulumi.StringMap{
		"app": pulumi.String(name),
	}

	// ConfigMap with metadata from IaC
	configMap, err := corev1.NewConfigMap(ctx, fmt.Sprintf("%s-metadata", name), &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Namespace: ns,
			Labels:    labels,
		},
		Data: pulumi.StringMap{
			"DECLARED_REGION":        args.Region,
			"DECLARED_INSTANCE_TYPE": args.InstanceType,
			"DECLARED_REPLICAS": replicas.ToIntPtrOutput().ApplyT(func(r *int) string {
				if r != nil {
					return fmt.Sprintf("%d", *r)
				}
				return "1"
			}).(pulumi.StringOutput),
			"DECLARED_SERVICE_TYPE": args.ServiceType,
			"DECLARED_DNS":          args.DNS,
			"DECLARED_TLS":          args.TLS,
			"DECLARED_NLB":          args.NLB,
			"DECLARED_NONROOT":      pulumi.String("true"),
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}
	component.ConfigMap = configMap

	// Deployment
	dep, err := appsv1.NewDeployment(ctx, fmt.Sprintf("%s-dep", name), &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Namespace: ns,
			Labels:    labels,
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: replicas,
			Strategy: &appsv1.DeploymentStrategyArgs{
				Type: pulumi.String("RollingUpdate"),
				RollingUpdate: &appsv1.RollingUpdateDeploymentArgs{
					MaxSurge:       pulumi.Int(1),
					MaxUnavailable: pulumi.Int(0),
				},
			},
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: labels,
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: labels,
					Annotations: pulumi.StringMap{
						"pulumi.com/configHash": args.DisplayValue.ToStringOutput().ApplyT(func(v string) string {
							return fmt.Sprintf("%d", len(v))
						}).(pulumi.StringOutput),
					},
				},
				Spec: &corev1.PodSpecArgs{
					SecurityContext: &corev1.PodSecurityContextArgs{
						RunAsNonRoot: pulumi.BoolPtr(true),
						RunAsUser:    pulumi.IntPtr(1001),
						FsGroup:      pulumi.IntPtr(1001),
					},
					TerminationGracePeriodSeconds: pulumi.IntPtr(30),
					Volumes: corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name: pulumi.String("tmp"),
							EmptyDir: &corev1.EmptyDirVolumeSourceArgs{},
						},
					},
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:  pulumi.String("app"),
							Image: args.Image, // required
							Ports: corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{
									ContainerPort: pulumi.Int(3000),
								},
							},
							Env: corev1.EnvVarArray{
								&corev1.EnvVarArgs{
									Name:  pulumi.String("DISPLAY_VALUE"),
									Value: args.DisplayValue, // required
								},
								// Downward API - inject runtime pod metadata
								&corev1.EnvVarArgs{
									Name: pulumi.String("KUBERNETES_NAMESPACE"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										FieldRef: &corev1.ObjectFieldSelectorArgs{
											FieldPath: pulumi.String("metadata.namespace"),
										},
									},
								},
								&corev1.EnvVarArgs{
									Name: pulumi.String("POD_NAME"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										FieldRef: &corev1.ObjectFieldSelectorArgs{
											FieldPath: pulumi.String("metadata.name"),
										},
									},
								},
								&corev1.EnvVarArgs{
									Name: pulumi.String("NODE_NAME"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										FieldRef: &corev1.ObjectFieldSelectorArgs{
											FieldPath: pulumi.String("spec.nodeName"),
										},
									},
								},
								&corev1.EnvVarArgs{
									Name: pulumi.String("POD_IP"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										FieldRef: &corev1.ObjectFieldSelectorArgs{
											FieldPath: pulumi.String("status.podIP"),
										},
									},
								},
							},
							EnvFrom: corev1.EnvFromSourceArray{
								&corev1.EnvFromSourceArgs{
									ConfigMapRef: &corev1.ConfigMapEnvSourceArgs{
										Name: configMap.Metadata.Name(),
									},
								},
							},
							Resources: &corev1.ResourceRequirementsArgs{
								Requests: pulumi.StringMap{
									"cpu":    pulumi.String("100m"),
									"memory": pulumi.String("128Mi"),
								},
								Limits: pulumi.StringMap{
									"cpu":    pulumi.String("200m"),
									"memory": pulumi.String("256Mi"),
								},
							},
							VolumeMounts: corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("tmp"),
									MountPath: pulumi.String("/tmp"),
								},
							},
							SecurityContext: &corev1.SecurityContextArgs{
								AllowPrivilegeEscalation: pulumi.BoolPtr(false),
								RunAsNonRoot:             pulumi.BoolPtr(true),
								RunAsUser:                pulumi.IntPtr(1001),
								Capabilities: &corev1.CapabilitiesArgs{
									Drop: pulumi.StringArray{
										pulumi.String("ALL"),
									},
								},
								ReadOnlyRootFilesystem: pulumi.BoolPtr(true),
							},
							LivenessProbe: &corev1.ProbeArgs{
								HttpGet: &corev1.HTTPGetActionArgs{
									Path: pulumi.String("/healthz"),
									Port: pulumi.Int(3000),
								},
								InitialDelaySeconds: pulumi.Int(10),
								PeriodSeconds:       pulumi.Int(10),
								TimeoutSeconds:      pulumi.Int(2),
								FailureThreshold:    pulumi.Int(3),
							},
							ReadinessProbe: &corev1.ProbeArgs{
								HttpGet: &corev1.HTTPGetActionArgs{
									Path: pulumi.String("/readyz"),
									Port: pulumi.Int(3000),
								},
								InitialDelaySeconds: pulumi.Int(5),
								PeriodSeconds:       pulumi.Int(5),
								TimeoutSeconds:      pulumi.Int(2),
								FailureThreshold:    pulumi.Int(2),
							},
						},
					},
				},
			},
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err
	}
	component.Deployment = dep

	// Service
	svc, err := corev1.NewService(ctx, fmt.Sprintf("%s-svc", name), &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Namespace: ns,
			Labels:    labels,
		},
		Spec: &corev1.ServiceSpecArgs{
			Type:     pulumi.String("ClusterIP"),
			Selector: labels,
			Ports: corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Port:       pulumi.Int(80),
					TargetPort: pulumi.Int(3000),
				},
			},
		},
	}, pulumi.Parent(component))
	if err != nil {
		return nil, err     
	}
	component.Service = svc

	// Outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"serviceName":    svc.Metadata.Name(),
		"deploymentName": dep.Metadata.Name(),
	}); err != nil {
		return nil, err
	}
	return component, nil
}
