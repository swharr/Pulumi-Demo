package webapp

import (
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)


type WebAppArgs struct {
Image pulumi.StringInput
DisplayValue pulumi.StringInput
Replicas pulumi.IntPtrInput
Namespace pulumi.StringPtrInput
}


type WebApp struct {
pulumi.ResourceState


Service *v1.Service
Deployment *appsv1.Deployment
}