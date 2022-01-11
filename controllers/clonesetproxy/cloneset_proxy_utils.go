package clonesetproxy

import (
	"context"
	"fmt"
	kruiseapps "github.com/openkruise/kruise-api/apps/v1alpha1"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"tutorial.kubebuilder.io/project/utils"
)

const (
	LabelCloneSetMode = "cloneset.asi/mode"
	CloneSetASI       = "asi"
)

func translateToCloneSet(reader client.Reader, src *apps.StatefulSet, dst *kruiseapps.CloneSet) {
	// TODO
	srcUpgradeStrategy := utils.GetUpgradeStrategy(src)

	if err := controllerutil.SetControllerReference(src, dst, localSchema); err != nil {
		klog.Errorf("failed to set owner refer for during translate to cloneset")
	}
	if dst.Labels == nil {
		dst.Labels = map[string]string{}
	}
	if dst.Annotations == nil {
		dst.Annotations = map[string]string{}
	}
	dst.Labels[LabelCloneSetMode] = CloneSetASI
	// generation 是在这里被设置的, cloneset 自己会改这个值吗?
	dst.Annotations[AnnotationCloneSetProxyGeneration] = fmt.Sprintf("%d", src.GetGeneration())
	for k, v := range src.GetLabels() {
		dst.Labels[k] = v
	}

	// 设置 annotation, annotation 有一个全局配置, 分别是 inheritedAnnotations & translateAnnotations
	// 设置 podUPgradeTimeout 到 dst.annotation 中
	// 设置 batchAdoption 问题到 dst.Labels()

	// dst.Spec.ScaleStrategy.PodsToDelete 从 src.PodsToDelete 和 actionPods 中获取
	// 根据字段的变化, 判断 revision 是否会 change, 如果 revisoin 不会change, 而改变了升级策略,
	// 那么升级策略是不会发生变化的

	// 设置 partition 到 dst.Spec.updateStrategy.Partition
	// src.annotttion get maxSurge -> dst.SPec.updateStrategy.MaxSurge
	// 为什么都已经有 src.upgradeStrategy, 还得从annotation 获取?
	// 接下来, 设置 maxUnavaiable 到 dst.spec.updateStrategy.MaxUnavailable 中
	// 从 src.annotation 中设置 scatter 策略, 这一点不懂

	// 处理 pre-annotation lifecyle pre delete
	// 下面是一些简单字段的赋值
	dst.Spec.Replicas = src.Spec.Replicas
	dst.Spec.Selector = src.Spec.Selector
	dst.Spec.Template = src.Spec.Template
	dst.Spec.VolumeClaimTemplates = src.Spec.VolumeClaimTemplates
	dst.Spec.RevisionHistoryLimit = src.Spec.RevisionHistoryLimit

	// copy minReadySeconds str

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

type UpgradeStrategyType string

type UpgradeStrategy struct {
	Type UpgradeStrategyType
}
