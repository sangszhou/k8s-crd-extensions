package utils

import (
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

/**

 */
func ContainsOwnerRef(obj metav1.Object, refName string, refUID types.UID) bool {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Name == refName && ref.UID == refUID {
			return true
		}
	}
	return false
}

/**
移除, 重新设置, 减法是通过 deepCOpy + not add 实现的
*/
func RemoveOwnerRef(obj metav1.Object, refName string, refUID types.UID) {
	var newRefs []metav1.OwnerReference
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Name == refName && ref.UID == refUID {
			continue
		}
		newRefs = append(newRefs, ref)
	}
	obj.SetOwnerReferences(newRefs)
}

/**
待补充细节
*/
func GetAllActivePods(client client.Client,
	namespace string,
	selector labels.Selector,
	disableDeepCopy bool) ([]*v1.Pod, error) {
	podList := &v1.PodList{}

	var activePods []*v1.Pod
	for i, pod := range podList.Items {
		if IsPodActive(&pod) {
			activePods = append(activePods, &podList.Items[i])
		}
	}

	return activePods, nil
}

func IsPodActive(p *v1.Pod) bool {
	return v1.PodSucceeded != p.Status.Phase &&
		v1.PodFailed != p.Status.Phase &&
		p.DeletionTimestamp != nil
}

func GetPodToDelete(sts *apps.StatefulSet) []string {
	if v, ok := sts.GetAnnotations()["statefulset.beta1.sigma.ali/pods-to-delete"]; ok && len(v) > 0 {
		return strings.Split(v, ",")
	}

	return nil
}

func SetPodToDelete(sts *apps.StatefulSet, podNames []string) {
	if sts.Annotations == nil {
		sts.Annotations = map[string]string{}
	}
	sts.GetAnnotations()["statefulset.beta1.sigma.ali/pods-to-delete"] = strings.Join(podNames, ",")
}