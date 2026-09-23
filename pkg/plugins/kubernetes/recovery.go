package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/consize-oss/consize/pkg/plugin"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type snapshot struct {
	Context   string                                 `json:"context"`
	UID       string                                 `json:"uid"`
	Resources map[string]corev1.ResourceRequirements `json:"resources"`
}

type snapshotPatcher interface {
	PrepareSnapshot(context.Context, string, string, PatchDiff) (snapshot, snapshot, error)
	InspectSnapshot(context.Context, string, string, snapshot, snapshot) (string, error)
	ApplySnapshot(context.Context, string, string, snapshot, snapshot, bool) error
	ReadySnapshot(context.Context, string, string) error
	PreflightSnapshot(context.Context, string, string, snapshot, snapshot) ([]plugin.PreflightCheck, error)
}

func capture(dep *appsv1.Deployment) snapshot {
	spec := dep.Spec.DeepCopy()
	for i := range spec.Template.Spec.Containers {
		spec.Template.Spec.Containers[i].Resources = corev1.ResourceRequirements{}
	}
	data, _ := json.Marshal(spec)
	s := snapshot{Context: string(data), UID: string(dep.UID), Resources: map[string]corev1.ResourceRequirements{}}
	for _, c := range dep.Spec.Template.Spec.Containers {
		s.Resources[c.Name] = *c.Resources.DeepCopy()
	}
	return s
}

func (k *K8sPatcher) PrepareSnapshot(ctx context.Context, ns, name string, diff PatchDiff) (snapshot, snapshot, error) {
	dep, err := k.client.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return snapshot{}, snapshot{}, err
	}
	var request, limit int64
	for _, c := range dep.Spec.Template.Spec.Containers {
		r, l := fieldValues(diff.Resource, c.Resources.Requests, c.Resources.Limits)
		request += r
		limit += l
	}
	if request != diff.CurrentReq || limit != diff.CurrentLimit {
		return snapshot{}, snapshot{}, errors.New("deployment changed during preparation")
	}
	original := capture(dep)
	if !mutateResources(dep.Spec.Template.Spec.Containers, diff) {
		return snapshot{}, snapshot{}, errors.New("no resources to mutate")
	}
	var proposedReq, proposedLim int64
	for _, c := range dep.Spec.Template.Spec.Containers {
		r, l := fieldValues(diff.Resource, c.Resources.Requests, c.Resources.Limits)
		if r < 0 || l < 0 || (l > 0 && r > l) {
			return snapshot{}, snapshot{}, errors.New("invalid per-container request or limit")
		}
		proposedReq += r
		proposedLim += l
	}
	if proposedReq != diff.ProposedReq || proposedLim != diff.ProposedLimit {
		return snapshot{}, snapshot{}, errors.New("per-container allocation cannot represent requested totals")
	}
	return original, capture(dep), nil
}

func equalSnapshot(a, b snapshot) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return reflect.DeepEqual(left, right)
}

func (k *K8sPatcher) InspectSnapshot(ctx context.Context, ns, name string, original, target snapshot) (string, error) {
	dep, err := k.client.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	actual := capture(dep)
	if equalSnapshot(actual, original) {
		return plugin.StateOriginal, nil
	}
	if equalSnapshot(actual, target) {
		return plugin.StateApplied, nil
	}
	return plugin.StateDrifted, nil
}

func (k *K8sPatcher) ApplySnapshot(ctx context.Context, ns, name string, expected, desired snapshot, rollback bool) error {
	for attempt := 0; attempt < 3; attempt++ {
		dep, err := k.client.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		actual := capture(dep)
		if equalSnapshot(actual, desired) {
			return nil
		}
		if !equalSnapshot(actual, expected) {
			return errors.New("deployment drifted; refusing to overwrite external changes")
		}
		if !rollback {
			checks, err := k.PreflightSnapshot(ctx, ns, name, expected, desired)
			if err != nil {
				return err
			}
			if err = plugin.RequirePreflight(checks); err != nil {
				return err
			}
		}
		for i := range dep.Spec.Template.Spec.Containers {
			resources, ok := desired.Resources[dep.Spec.Template.Spec.Containers[i].Name]
			if !ok {
				return errors.New("container identity changed")
			}
			dep.Spec.Template.Spec.Containers[i].Resources = *resources.DeepCopy()
		}
		_, err = k.client.AppsV1().Deployments(ns).Update(ctx, dep, metav1.UpdateOptions{})
		if err == nil {
			return nil
		}
		if !apierrors.IsConflict(err) {
			return err
		}
	}
	return errors.New("deployment update conflicted repeatedly")
}

func (k *K8sPatcher) ReadySnapshot(ctx context.Context, ns, name string) error {
	dep, err := k.client.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	desired := int32(1)
	if dep.Spec.Replicas != nil {
		desired = *dep.Spec.Replicas
	}
	if desired == 0 {
		return errors.New("deployment is scaled to zero")
	}
	if dep.Status.ObservedGeneration < dep.Generation || dep.Status.UpdatedReplicas != desired || dep.Status.AvailableReplicas < desired || dep.Status.ReadyReplicas < desired || dep.Status.Replicas != desired {
		return errors.New("deployment rollout is not healthy yet")
	}
	return nil
}

func decodeSnapshot(value any) (snapshot, error) {
	var s snapshot
	data, err := json.Marshal(value)
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	if err == nil && (s.UID == "" || s.Context == "" || len(s.Resources) == 0) {
		err = errors.New("missing immutable deployment identity or resource snapshot")
	}
	return s, err
}

func (p *Plugin) snapshots(plan plugin.ActionPlan) (snapshotPatcher, string, string, snapshot, snapshot, error) {
	patcher, ok := p.patcher.(snapshotPatcher)
	if !ok {
		return nil, "", "", snapshot{}, snapshot{}, errors.New("patcher does not support recovery")
	}
	ns, _ := plan.Diff["namespace"].(string)
	name, _ := plan.Diff["name"].(string)
	original, err := decodeSnapshot(plan.Diff["original"])
	if err != nil {
		return nil, "", "", original, snapshot{}, err
	}
	target, err := decodeSnapshot(plan.Diff["target"])
	if err != nil {
		return nil, "", "", original, target, err
	}
	if ns == "" || name == "" {
		err = errors.New("missing deployment identity")
	}
	return patcher, ns, name, original, target, err
}

func (p *Plugin) Inspect(ctx context.Context, plan plugin.ActionPlan) (string, error) {
	k, ns, name, original, target, err := p.snapshots(plan)
	if err != nil {
		return "", err
	}
	return k.InspectSnapshot(ctx, ns, name, original, target)
}
func (p *Plugin) Ready(ctx context.Context, plan plugin.ActionPlan) error {
	k, ns, name, _, _, err := p.snapshots(plan)
	if err != nil {
		return err
	}
	return k.ReadySnapshot(ctx, ns, name)
}
func (p *Plugin) Rollback(ctx context.Context, plan plugin.ActionPlan) (plugin.ActionResult, error) {
	return p.applySnapshot(ctx, plan, true)
}
func (p *Plugin) applySnapshot(ctx context.Context, plan plugin.ActionPlan, rollback bool) (plugin.ActionResult, error) {
	k, ns, name, original, target, err := p.snapshots(plan)
	if err != nil {
		return plugin.ActionResult{}, err
	}
	if rollback {
		original, target = target, original
	}
	if err := k.ApplySnapshot(ctx, ns, name, original, target, rollback); err != nil {
		return plugin.ActionResult{}, err
	}
	verb := "applied"
	if rollback {
		verb = "restored"
	}
	return plugin.ActionResult{PluginID: ID, ResourceID: plan.ResourceID, ActionType: plan.ActionType, Applied: true, Message: fmt.Sprintf("%s resource snapshot for %s/%s", verb, ns, name)}, nil
}
