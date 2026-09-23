package kubernetes

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/consize-oss/consize/pkg/plugin"
	"github.com/consize-oss/consize/pkg/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	quantity "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func TestLiveKubernetesApplyRollback(t *testing.T) {
	if os.Getenv("CONSIZE_LOCAL_INTEGRATION") != "1" {
		t.Skip("set CONSIZE_LOCAL_INTEGRATION=1 to test Docker Desktop only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{CurrentContext: "docker-desktop"}).ClientConfig()
	if err != nil {
		t.Fatal(err)
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ns := fmt.Sprintf("consize-safety-smoke-%d", time.Now().Unix())
	if _, err := client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 20*time.Second)
		defer stop()
		if err := client.CoreV1().Namespaces().Delete(cleanupCtx, ns, metav1.DeleteOptions{}); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	}()
	replicas := int32(1)
	labels := map[string]string{"app": "safety-smoke"}
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: ns}, Spec: appsv1.DeploymentSpec{Replicas: &replicas, Selector: &metav1.LabelSelector{MatchLabels: labels}, Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "nginx:1.27-alpine", Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceMemory: quantity.MustParse("512Mi")}, Limits: corev1.ResourceList{corev1.ResourceMemory: quantity.MustParse("1Gi")}}}}}}}}
	if _, err := client.AppsV1().Deployments(ns).Create(ctx, dep, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	patcher := NewK8sPatcherFromClient(client)
	p := NewWithPatcher(patcher)
	waitReady := func() {
		for {
			if err := patcher.ReadySnapshot(ctx, ns, "smoke"); err == nil {
				return
			}
			select {
			case <-ctx.Done():
				t.Fatal("rollout timeout")
			case <-time.After(2 * time.Second):
			}
		}
	}
	waitReady()
	plan, err := p.Plan(ctx, plugin.ActionInput{Resource: resource.Resource{ID: ns, Type: resource.TypeKubernetesDeployment, Metadata: map[string]any{"namespace": ns, "name": "smoke"}, CurrentState: map[string]any{"memory_request_bytes": int64(512 * 1024 * 1024), "memory_limit_bytes": int64(1024 * 1024 * 1024)}}, Parameters: map[string]any{"patch": PatchDiff{Resource: "memory", CurrentReq: 512 * 1024 * 1024, ProposedReq: 384 * 1024 * 1024, CurrentLimit: 1024 * 1024 * 1024, ProposedLimit: 1024 * 1024 * 1024}}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := p.Execute(ctx, plan); err != nil {
			t.Fatal(err)
		}
	}
	waitReady()
	if state, err := p.Inspect(ctx, plan); err != nil || state != plugin.StateApplied {
		t.Fatalf("apply: state=%s error=%v", state, err)
	}
	for i := 0; i < 2; i++ {
		if _, err := p.Rollback(ctx, plan); err != nil {
			t.Fatal(err)
		}
	}
	waitReady()
	if state, err := p.Inspect(ctx, plan); err != nil || state != plugin.StateOriginal {
		t.Fatalf("rollback: state=%s error=%v", state, err)
	}
	request, limit, err := patcher.ReadDeploymentResources(ctx, ns, "smoke", "memory")
	if err != nil || request != 512*1024*1024 || limit != 1024*1024*1024 {
		t.Fatalf("restoration request=%d limit=%d error=%v", request, limit, err)
	}
}
