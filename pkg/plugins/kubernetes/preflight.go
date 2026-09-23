package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/consize-oss/consize/pkg/plugin"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (p *Plugin) Preflight(ctx context.Context, plan plugin.ActionPlan) ([]plugin.PreflightCheck, error) {
	k, ns, name, original, target, err := p.snapshots(plan)
	if err != nil {
		return nil, err
	}
	return k.PreflightSnapshot(ctx, ns, name, original, target)
}

func podFromSnapshot(s snapshot) (corev1.PodSpec, error) {
	var spec appsv1.DeploymentSpec
	if err := json.Unmarshal([]byte(s.Context), &spec); err != nil {
		return corev1.PodSpec{}, err
	}
	if len(spec.Template.Spec.Containers) == 0 {
		return corev1.PodSpec{}, fmt.Errorf("snapshot has no containers")
	}
	for i := range spec.Template.Spec.Containers {
		r, ok := s.Resources[spec.Template.Spec.Containers[i].Name]
		if !ok {
			return corev1.PodSpec{}, fmt.Errorf("snapshot is missing container resources")
		}
		spec.Template.Spec.Containers[i].Resources = *r.DeepCopy()
	}
	return spec.Template.Spec, nil
}

// Container-level QoS follows Kubernetes's CPU/memory rules, including init
// containers. Pod-level budgets are blocked rather than guessed across feature gates.
func qosClass(spec corev1.PodSpec) (corev1.PodQOSClass, error) {
	if spec.Resources != nil {
		return "", fmt.Errorf("pod-level resource budgets require a separate supported QoS model")
	}
	anyResource, guaranteed := false, true
	containers := append(append([]corev1.Container{}, spec.Containers...), spec.InitContainers...)
	if len(containers) == 0 {
		return "", fmt.Errorf("no containers to classify")
	}
	for _, c := range containers {
		for _, key := range []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory} {
			request, requestOK := c.Resources.Requests[key]
			limit, limitOK := c.Resources.Limits[key]
			if request.Sign() > 0 || limit.Sign() > 0 {
				anyResource = true
			}
			if !requestOK || !limitOK || request.Sign() <= 0 || limit.Sign() <= 0 || request.Cmp(limit) != 0 {
				guaranteed = false
			}
		}
	}
	if !anyResource {
		return corev1.PodQOSBestEffort, nil
	}
	if guaranteed {
		return corev1.PodQOSGuaranteed, nil
	}
	return corev1.PodQOSBurstable, nil
}

func check(id, status, message string, details map[string]any) plugin.PreflightCheck {
	return plugin.PreflightCheck{ID: id, Status: status, Message: message, Details: details}
}

func (k *K8sPatcher) PreflightSnapshot(ctx context.Context, ns, name string, original, target snapshot) ([]plugin.PreflightCheck, error) {
	before, err := podFromSnapshot(original)
	if err != nil {
		return nil, err
	}
	after, err := podFromSnapshot(target)
	if err != nil {
		return nil, err
	}
	dep, err := k.client.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return []plugin.PreflightCheck{check("workload", "unknown", "Cannot inspect live Deployment: "+err.Error(), nil)}, nil
	}
	actual := capture(dep)
	if !equalSnapshot(actual, original) && !equalSnapshot(actual, target) {
		return []plugin.PreflightCheck{check("workload", "unknown", "Deployment changed since the captured plan", nil)}, nil
	}
	results := []plugin.PreflightCheck{}
	oldQoS, oldErr := qosClass(before)
	newQoS, newErr := qosClass(after)
	details := map[string]any{"before": string(oldQoS), "after": string(newQoS)}
	if oldErr != nil || newErr != nil {
		results = append(results, check("qos", "unknown", fmt.Sprintf("Cannot establish QoS classification: %v; %v", oldErr, newErr), details))
	} else {
		rank := map[corev1.PodQOSClass]int{corev1.PodQOSBestEffort: 0, corev1.PodQOSBurstable: 1, corev1.PodQOSGuaranteed: 2}
		if rank[newQoS] < rank[oldQoS] {
			results = append(results, check("qos", "blocked", "Request reduction would downgrade Pod QoS", details))
		} else {
			results = append(results, check("qos", "passed", "No Pod QoS downgrade", details))
		}
	}
	changed := map[corev1.ResourceName]map[string]bool{}
	for _, key := range []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory} {
		changed[key] = map[string]bool{}
		for _, c := range before.Containers {
			old := c.Resources.Requests[key]
			next := target.Resources[c.Name].Requests[key]
			if old.Cmp(next) != 0 {
				changed[key][c.Name] = true
			}
		}
	}
	hpas, err := k.client.AutoscalingV2().HorizontalPodAutoscalers(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		results = append(results, check("hpa", "unknown", "Cannot inspect HPA controllers: "+err.Error(), nil))
	} else {
		conflicts, matching := []string{}, []string{}
		for _, hpa := range hpas.Items {
			if !deploymentTarget(hpa.Spec.ScaleTargetRef.APIVersion, hpa.Spec.ScaleTargetRef.Kind, hpa.Spec.ScaleTargetRef.Name, name) {
				continue
			}
			matching = append(matching, hpa.Name)
			metrics := hpa.Spec.Metrics
			if len(metrics) == 0 {
				metrics = []autoscalingv2.MetricSpec{{Type: autoscalingv2.ResourceMetricSourceType, Resource: &autoscalingv2.ResourceMetricSource{Name: corev1.ResourceCPU, Target: autoscalingv2.MetricTarget{Type: autoscalingv2.UtilizationMetricType}}}}
			}
			for _, metric := range metrics {
				if metric.Resource != nil && metric.Resource.Target.Type == autoscalingv2.UtilizationMetricType && len(changed[metric.Resource.Name]) > 0 {
					conflicts = append(conflicts, hpa.Name+": "+string(metric.Resource.Name)+" utilization depends on the changed request")
				}
				if metric.ContainerResource != nil && metric.ContainerResource.Target.Type == autoscalingv2.UtilizationMetricType && changed[metric.ContainerResource.Name][metric.ContainerResource.Container] {
					conflicts = append(conflicts, hpa.Name+": container utilization depends on the changed request")
				}
			}
		}
		status, message := "passed", "No request-dependent HPA conflict detected"
		if len(conflicts) > 0 {
			status = "blocked"
			message = strings.Join(conflicts, "; ")
		}
		results = append(results, check("hpa", status, message, map[string]any{"controllers": matching}))
	}
	results = append(results, k.vpaCheck(ctx, ns, name))
	return results, nil
}

func deploymentTarget(apiVersion, kind, target, name string) bool {
	if kind != "Deployment" || target != name {
		return false
	}
	if apiVersion == "" {
		return true
	}
	gv, err := schema.ParseGroupVersion(apiVersion)
	return err == nil && gv.Group == "apps"
}

func (k *K8sPatcher) vpaCheck(ctx context.Context, ns, name string) plugin.PreflightCheck {
	groups, err := k.client.Discovery().ServerGroups()
	if err != nil {
		return check("vpa", "unknown", "Cannot discover VPA APIs: "+err.Error(), nil)
	}
	var resource *schema.GroupVersionResource
	for _, group := range groups.Groups {
		if group.Name != "autoscaling.k8s.io" {
			continue
		}
		for _, version := range group.Versions {
			resources, err := k.client.Discovery().ServerResourcesForGroupVersion(version.GroupVersion)
			if err != nil {
				return check("vpa", "unknown", "Cannot inspect advertised VPA API: "+err.Error(), nil)
			}
			for _, r := range resources.APIResources {
				if r.Name == "verticalpodautoscalers" {
					gv, err := schema.ParseGroupVersion(version.GroupVersion)
					if err != nil {
						return check("vpa", "unknown", "Malformed VPA API version", nil)
					}
					gvr := gv.WithResource(r.Name)
					resource = &gvr
					break
				}
			}
			if resource != nil {
				break
			}
		}
		if resource != nil {
			break
		}
	}
	if resource == nil {
		return check("vpa", "passed", "API discovery reports no installed VPA resource", nil)
	}
	if k.dynamic == nil {
		return check("vpa", "unknown", "VPA API is installed but no dynamic inspection client is available", nil)
	}
	list, err := k.dynamic.Resource(*resource).Namespace(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return check("vpa", "unknown", "Cannot list VPA controllers: "+err.Error(), nil)
	}
	matching, active := []string{}, []string{}
	for _, vpa := range list.Items {
		ref, found, err := unstructured.NestedMap(vpa.Object, "spec", "targetRef")
		if err != nil || !found {
			return check("vpa", "unknown", "VPA has an unreadable target reference", nil)
		}
		kind, _ := ref["kind"].(string)
		target, _ := ref["name"].(string)
		version, _ := ref["apiVersion"].(string)
		if kind == "" || target == "" {
			return check("vpa", "unknown", "VPA target reference is incomplete", nil)
		}
		if !deploymentTarget(version, kind, target, name) {
			continue
		}
		matching = append(matching, vpa.GetName())
		mode, found, err := unstructured.NestedString(vpa.Object, "spec", "updatePolicy", "updateMode")
		if err != nil {
			return check("vpa", "unknown", "VPA update mode is unreadable", nil)
		}
		if !found {
			mode = "Auto"
		}
		switch mode {
		case "Off":
		case "Auto", "Initial", "Recreate", "InPlaceOrRecreate", "InPlace":
			active = append(active, vpa.GetName()+" ("+mode+")")
		default:
			return check("vpa", "unknown", "Unsupported VPA update mode: "+mode, nil)
		}
	}
	if len(active) > 0 {
		return check("vpa", "blocked", "VPA may override requests during rollout: "+strings.Join(active, ", "), map[string]any{"controllers": matching})
	}
	return check("vpa", "passed", "No actively managing VPA targets this Deployment", map[string]any{"controllers": matching})
}
