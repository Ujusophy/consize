package kubernetes

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestK8sPatcherPatchesDeploymentResources(t *testing.T) {
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "checkout-api", Namespace: "prod"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "app",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceMemory: *k8sresource.NewQuantity(1024, k8sresource.BinarySI)},
							Limits:   corev1.ResourceList{corev1.ResourceMemory: *k8sresource.NewQuantity(2048, k8sresource.BinarySI)},
						},
					}},
				},
			},
		},
	})
	patcher := NewK8sPatcherFromClient(client)
	if err := patcher.PatchDeployment(context.Background(), "prod", "checkout-api", PatchDiff{
		Resource:      "memory",
		CurrentReq:    1024,
		ProposedReq:   768,
		CurrentLimit:  2048,
		ProposedLimit: 1536,
	}); err != nil {
		t.Fatal(err)
	}
	req, lim, err := patcher.ReadDeploymentResources(context.Background(), "prod", "checkout-api", "memory")
	if err != nil {
		t.Fatal(err)
	}
	if req != 768 || lim != 1536 {
		t.Fatalf("resources = request %d limit %d, want request 768 limit 1536", req, lim)
	}
}
