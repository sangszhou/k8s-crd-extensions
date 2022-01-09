package clonesetproxy

import (
	"context"
	kruiseapps "github.com/openkruise/kruise-api/apps/v1alpha1"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"tutorial.kubebuilder.io/project/utils"
)

func translateToCloneSet(reader client.Reader, src *apps.StatefulSet, dst *kruiseapps.CloneSet) {
	// TODO
}

func convertCloneSetConditionToStatefulSetCondition(condition []kruiseapps.CloneSetCondition) []apps.StatefulSetCondition {
	return nil
}

/**
虽然 pod 的 reference 变了, 但是还是可以根据 label 和 podName 找到他的
*/
func filterActivePods(reader client.Reader, ns string, podName []string) []string {
	ret := sets.NewString()
	for _, name := range podName {
		pod := v1.Pod{}
		if err := reader.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: name}, &pod); err != nil {
			if !errors.IsNotFound(err) {
				klog.Warningf("failed to get pod")
			}
			continue
		}
		if !utils.IsPodActive(&pod) {
			continue
		}
		ret.Insert(name)
	}
	return ret.List()
}
