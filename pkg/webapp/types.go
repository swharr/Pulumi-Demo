package webapp

import (
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)


type WebAppArgs struct {
	Image        pulumi.StringInput
	DisplayValue pulumi.StringInput
	Replicas     pulumi.IntPtrInput
	Namespace    pulumi.StringPtrInput

	// IaC metadata for ConfigMap
	Region       pulumi.StringInput
	InstanceType pulumi.StringInput
	ServiceType  pulumi.StringInput
	DNS          pulumi.StringInput
	TLS          pulumi.StringInput
	NLB          pulumi.StringInput
}


type WebApp struct {
	pulumi.ResourceState

	ConfigMap  *v1.ConfigMap
	Service    *v1.Service
	Deployment *appsv1.Deployment
}