package clonesetproxy

import (
	"context"
	"fmt"
	kruiseapps "github.com/openkruise/kruise-api/apps/v1alpha1"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/controller/history"
	"k8s.io/kubernetes/pkg/util/slice"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strconv"
	"tutorial.kubebuilder.io/project/utils"
	"tutorial.kubebuilder.io/project/utils/kubeclient"
)

const (
	LabelInPlaceSetProxy              = "inplaceset.sigma.ali/proxy"
	ProxyToCloneSet                   = "CloneSet"
	AnnotationCloneProxyInitializing  = "cloneset.asi/proxy-initializing"
	AnnotationCloneSetProxyGeneration = "cloneset.asi/proxy-generation"

	AnnotationForbidAutoProxy = "cloneset.asi.sigma.ali/forbid-auto-proxy"

	finalizerCloneSetProxy = "alibabacloud.com/proxy-to-cloneset"
)

var (
	cloneSetKind    = kruiseapps.SchemeGroupVersion.WithKind("CloneSet")
	statefulSetKind = apps.SchemeGroupVersion.WithKind("StatefulSet")

	// 这个 local schema 是干啥的
	localSchema = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(localSchema)
	// appsv1beta1 是 ak8s-ee 的内容, 外网无法引入
	//_ = appsv1beta1.
}

func Add(mgr manager.Manager) error {
	// register field index, 需要 utils 包的支持
	//if err := fieldindex.

	if installed, err := kubeclient.IsResourceInstalled(cloneSetKind); err != nil {
		klog.Warningf("cloneset not installed 1")
		return nil
	} else if !installed {
		klog.Warningf("cloneset not installed 2")
		return nil
	}

	// 包括 ak8s.eee 的内容, 暂时跳过
	//r := &reconcile.Reconciler{
	//	Client : mgr.GetClient(),
	//	recorder:
	//}

}

type Reconciler struct {
	client.Client

	controllerHistory history.Interface
	recorder          record.EventRecorder
}

// 这里没看到这个赋值是啥意思
var _ reconcile.Reconciler = &Reconciler{}

/**
添加一个 controller
*/
func add(mgr manager.Manager, r *Reconciler) error {
	c, err := controller.New("cloneset-proxy-controller",
		mgr,
		controller.Options{Reconciler: r, MaxConcurrentReconciles: 20})

	if err != nil {
		klog.Errorf("failed to create controller")
		return err
	}

	err = c.Watch(&source.Kind{Type: &kruiseapps.CloneSet{}},
		&handler.EnqueueRequestForObject{}, predicate.Funcs{
			CreateFunc: func(event event.CreateEvent) bool {
				cloneset := event.Object.(*kruiseapps.CloneSet)
				return cloneset.GetLabels()[LabelInPlaceSetProxy] == ProxyToCloneSet
			},
			UpdateFunc: func(event event.UpdateEvent) bool {
				cloneset := event.ObjectNew.(*kruiseapps.CloneSet)
				return cloneset.GetLabels()[LabelInPlaceSetProxy] == ProxyToCloneSet
			},
			DeleteFunc: func(event event.DeleteEvent) bool {
				cloneset := event.Object.(*kruiseapps.CloneSet)
				return cloneset.GetLabels()[LabelInPlaceSetProxy] == ProxyToCloneSet
			},
			GenericFunc: func(event event.GenericEvent) bool {
				cloneset := event.Object.(*kruiseapps.CloneSet)
				return cloneset.GetLabels()[LabelInPlaceSetProxy] == ProxyToCloneSet
			},
		})

	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &apps.StatefulSet{}},
		&handler.EnqueueRequestForObject{}, predicate.Funcs{
			CreateFunc: func(event event.CreateEvent) bool {
				sts := event.Object.(*apps.StatefulSet)
				return sts.GetLabels()[LabelInPlaceSetProxy] == ProxyToCloneSet
			},
			UpdateFunc: func(event event.UpdateEvent) bool {
				sts := event.ObjectNew.(*apps.StatefulSet)
				return sts.GetLabels()[LabelInPlaceSetProxy] == ProxyToCloneSet
			},
			DeleteFunc: func(event event.DeleteEvent) bool {
				sts := event.Object.(*apps.StatefulSet)
				return sts.GetLabels()[LabelInPlaceSetProxy] == ProxyToCloneSet
			},
			GenericFunc: func(event event.GenericEvent) bool {
				sts := event.Object.(*apps.StatefulSet)
				return sts.GetLabels()[LabelInPlaceSetProxy] == ProxyToCloneSet
			},
		})

	return nil
}

/**
需要各种权限, 需要补齐
*/
func (r *Reconciler) Reconcile(req reconcile.Request) (retRes reconcile.Result, err error) {
	srcSet, err := r.getSts(req)
	if err != nil {
		klog.Errorf("Not found sts")
		return reconcile.Result{}, nil
	}

	if srcSet.GetDeletionTimestamp() != nil {
		return reconcile.Result{}, nil
	}

	if srcSet.GetLabels()[LabelInPlaceSetProxy] != ProxyToCloneSet {
		klog.Warningf("skip reconcile cloneset for no proxy label ")
	}

	// 查询对应的 cloneset 是否存在, 如果不存在的话, 创建之
	cloneSet := &kruiseapps.CloneSet{}
	if err := r.Get(context.TODO(), req.NamespacedName, cloneSet); err != nil {
		if !errors.IsNotFound(err) {
			if err := r.createCloneSet(srcSet); err != nil {
			}
			return reconcile.Result{}, err
		}
		// 靠重新调度, 而不是直接往下走, 有什么好处吗?
		return reconcile.Result{}, nil
	}

	if migrateCounter, err := r.migrateChildren(srcSet, cloneSet); err != nil {
		return reconcile.Result{}, err
	} else if migrateCounter > 0 {
		r.recorder.Event(srcSet, v1.EventTypeNormal,
			"SuccessfullMigrateProxy",
			"Successfully migrate for proxy")
	}

	// 按照 source workload 更新 cloneset
	if err := r.updateCloneSet(srcSet, cloneSet); err != nil {
		return reconcile.Result{}, err
	}

	// 根据 cloneset status 反向更新 source status
	if err := r.updateSourceStatus(srcSet, cloneSet); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.truncatePodsToDelete(srcSet); err != nil {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil

}

/**
这里需要对 statefulset, deployment 做一层抽象, 但是
*/
func (r *Reconciler) getSts(req reconcile.Request) (set *apps.StatefulSet, err error) {
	sts := apps.StatefulSet{}
	if err := r.Get(context.TODO(), req.NamespacedName, &sts); err == nil {
		return &sts, nil
	} else if !errors.IsNotFound(err) {
		return nil, err
	}

	return nil, nil
}

func (r *Reconciler) createCloneSet(src *apps.StatefulSet) error {
	// get and set global config from k8s configMap. 这里因为不好配置和测试, 就跳过了

	cloneSet := &kruiseapps.CloneSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: src.GetNamespace(),
			Annotations: map[string]string{
				AnnotationCloneProxyInitializing: "true",
			},
		},
	}

	translateToCloneSet(r.Client, src, cloneSet)

	if err := r.Create(context.TODO(), cloneSet); err != nil && !errors.IsAlreadyExists(err) {
		klog.Info("create from source failed")
		return err
	}

	klog.Info("create from source success")

	return nil
}

/**
还处于转移中
怎么保证 src 扩容出来的 pod 持续挂载到 dst 上呢?
*/
func (r *Reconciler) migrateChildren(src *apps.StatefulSet, dst *kruiseapps.CloneSet) (int, error) {
	// 不存在 proxy-initializing 说明完成了同步, 不需要进行迁移
	// > 在什么时候, 这个标记被清空呢?
	if _, ok := dst.Annotations[AnnotationCloneProxyInitializing]; !ok {
		return 0, nil
	}

	var migrateCounter int
	// 参考 sigma 的实现
	selector, err := metav1.LabelSelectorAsSelector(src.Spec.Selector)
	if err != nil {
		klog.Errorf("error parse selector")
		return migrateCounter, nil
	}

	/**
	这里是在干啊, controller revision 和 revision hash 是一个东西吗?
	ListControllerRevisions lists all ControllerRevisions matching selector and owned by parent or no other
	controller. If the returned error is nil the returned slice of ControllerRevisions is valid. If the
	returned error is not nil, the returned slice is not valid.
	*/
	revisions, err := r.controllerHistory.ListControllerRevisions(src, selector)
	if err != nil {
		return migrateCounter, nil
	}

	migratedRevisionNames := sets.NewString()
	defer func() {
		if migratedRevisionNames.Len() > 0 {
			klog.Info("has migrated todo numbers")
		}
	}()

	/**
	revision 代表了啥意思
	*/
	for _, cr := range revisions {
		// 这是确保 cr 是谁的孩子?
		// src 自己获取的 revision 还不是自己的孩子?
		if !utils.ContainsOwnerRef(cr, src.GetNamespace(), src.GetUID()) {
			continue
		}

		// cr 的 data 字段保存了这么多数据吗?
		oldOwner := metav1.GetControllerOf(cr)
		// 为什么 controller 还是老的controller 呢? controller 是干嘛的
		newOwner := metav1.OwnerReference{
			APIVersion:         cloneSetKind.GroupVersion().String(),
			Kind:               cloneSetKind.Kind,
			Name:               dst.Name,
			UID:                dst.UID,
			Controller:         oldOwner.Controller,
			BlockOwnerDeletion: oldOwner.BlockOwnerDeletion,
		}

		// 修改 cr 的reference 有啥用呢? cr 又不是一个资源. 如果 sts 的 revision 变了, 还需要重新修改吗?
		// 把 owner 转换成新的 dst, 但是 controller 还是保持不变?
		cr.OwnerReferences = append(cr.OwnerReferences, newOwner)

		if err := r.Update(context.TODO(), cr); err != nil {
			return migrateCounter, fmt.Errorf("err update controllerRevision")
		}

		migratedRevisionNames.Insert(cr.Name)
		migrateCounter++
	}

	pods, err := utils.GetAllActivePods(r.Client, src.GetNamespace(), selector, true)
	if err != nil {
		return migrateCounter, err
	}

	migratedPodNames := sets.NewString()
	defer func() {
		if migratedPodNames.Len() > 0 {
			klog.Info("has migrated pods for...")
		}
	}()

	// TODO 把 pod owner refer 都进行修改
}

func (r *Reconciler) updateCloneSet(src *apps.StatefulSet, dst *kruiseapps.CloneSet) error {
	original := dst.DeepCopy()
	// 所有 pod 都迁移到 cloneset 之后, 把创建加的 proxy-initializing 去掉
	if _, ok := dst.Annotations[AnnotationCloneProxyInitializing]; ok {
		delete(dst.Annotations, AnnotationCloneProxyInitializing)
	}

	if _, ok := dst.Annotations[AnnotationForbidAutoProxy]; ok {
		delete(dst.Annotations, AnnotationForbidAutoProxy)
	}

	// ToDo translateToCloneset
	translateToCloneSet(r.Client, src, dst)

	if reflect.DeepEqual(dst, original) {
		return nil
	}

	if err := r.Update(context.TODO(), dst); err != nil {
		return fmt.Errorf("error update cloneset")
	}

	klog.Infof("update cloneset from generation ")
	return nil
}

/**
根据 cloneset 的状态来反向更新
*/
func (r *Reconciler) updateSourceStatus(src *apps.StatefulSet, dst *kruiseapps.CloneSet) error {
	// 什么情况下, generation 会相等, generation 是谁控制更新的, sts, cloneset 和 deploment 都一样么
	if dst.Generation != dst.Status.ObservedGeneration {
		klog.Info("dst generation not equals")
		return nil
	}

	/**
	这个字段是干啥用的? generation 是否在其他地方也会被更新呢, 如果被更新的话, 那 genration status 改了
	也没用
	*/
	var proxyGen int64
	if v, ok := dst.Annotations[AnnotationCloneSetProxyGeneration]; !ok {
		return fmt.Errorf("not found proxy gen in cloneset annotations")
	} else {
		var err error
		if proxyGen, err = strconv.ParseInt(v, 10, 64); err != nil {
			return fmt.Errorf("error parse proxy-generation")
		}
	}

	currentRevision, err := r.getCurrentRevision(dst, dst.Status.UpdateRevision)
	if err != nil {
		return err
	}

	/**
	每个字段都是什么含义, 要搞清楚
	TODO
	1. condition 切换
	2. updatedReadyReplicas 需要填写
	*/
	newStatus := apps.StatefulSetStatus{
		ObservedGeneration: proxyGen,
		CurrentRevision:    currentRevision,
		UpdateRevision:     dst.Status.UpdateRevision,
		CollisionCount:     dst.Status.CollisionCount,
		// condition 一般都是记录些啥信息呢, 找几个例子看看
		Conditions: convertCloneSetConditionToStatefulSetCondition(dst.Status.Conditions),

		Replicas:        dst.Status.Replicas,
		ReadyReplicas:   dst.Status.ReadyReplicas,
		UpdatedReplicas: dst.Status.UpdatedReplicas,
		CurrentReplicas: dst.Status.Replicas - dst.Status.UpdatedReplicas,
	}

	// 这是干嘛的?
	if newStatus.CurrentRevision == newStatus.UpdateRevision {
		newStatus.CurrentReplicas = newStatus.UpdatedReplicas
	}

	// 同步 finalizer
	var finalizerChanged bool
	// 设置 finalizer
	if src.GetDeletionTimestamp() == nil &&
		!slice.ContainsString(src.GetFinalizers(), finalizerCloneSetProxy, nil) {
		finalizerChanged = true
		src.SetFinalizers(append(src.GetFinalizers(), finalizerCloneSetProxy))
	} else if src.GetDeletionTimestamp() != nil {
		finalizerChanged = true
		src.SetFinalizers(slice.RemoveString(src.GetFinalizers(), finalizerCloneSetProxy, nil))
	}

	if finalizerChanged {
		klog.Info("finalizer is true")
	}

	// update publish status
	src.Status = newStatus
	// 这里参数直接传递 src, 源代码是 src.update 吗?
	if err := r.Status().Update(context.TODO(), src); err != nil {
		return fmt.Errorf("error update status to utils.dumpJson")
	}

	klog.Infof("update status")

	return nil
}

/**
为啥查看 revision 的时候, 还得取比较 pod 和 controller 的 revision 呢?
pod 的 revision 和 controller 的 revisions 是怎样变化的? 以前这些问题, 都没问, 现在都会遇到的...
*/
func (r *Reconciler) getCurrentRevision(cloneSet *kruiseapps.CloneSet, updateRevision string) (string, error) {
	selector, _ := metav1.LabelSelectorAsSelector(cloneSet.Spec.Selector)
	filterPods, err := r.getActiveCloneSetPods(cloneSet)
	if err != nil {
		return "", err
	}

	podsRevisions := getPodRevisions(filterPods)

	// > 这是在干嘛
	revisions, err := r.controllerHistory.ListControllerRevisions(cloneSet, selector)
	if err != nil {
		return "", nil
	}

	// 这里的 revision 应该是从新到旧排序
	history.SortControllerRevisions(revisions)
	currentRevision := updateRevision
	// 这里应该是取最新的 revision
	for i := range revisions {
		if podsRevisions.Has(revisions[i].Name) {
			currentRevision = revisions[i].Name
			break
		}
	}

	return currentRevision, nil
}

/**
返回所有 active pod list, 对于 cloneset 下数量比较多的情况
*/
func (r *Reconciler) getActiveCloneSetPods(cloneSet *kruiseapps.CloneSet) ([]*v1.Pod, error) {
	podList := &v1.PodList{}

	// 还缺少一个 disable deep copy 的配置
	if err := r.List(context.TODO(), podList, client.InNamespace(cloneSet.Namespace),
		client.MatchingFields{"": string(cloneSet.UID)}); err != nil {
		return nil, err
	}

	var activePods []*v1.Pod
	// ignore inactive pods
	for i, pod := range podList.Items {
		if utils.IsPodActive(&pod) {
			activePods = append(activePods, &podList.Items[i])
		}
	}

	return activePods, nil
}

func getPodRevisions(pods []*v1.Pod) sets.String {
	revisions := sets.NewString()
	for _, p := range pods {
		revisions.Insert(p.Labels[apps.ControllerRevisionHashLabelKey])
	}
	return revisions
}

/**
从 annotation 字段, 删除那些已经不存在的 pod
这个比较好理解
*/
func (r *Reconciler) truncatePodsToDelete(srcSet *apps.StatefulSet) error {
	podsToDelete := utils.GetPodToDelete(srcSet)

	// 如果小于 tolerate 上限
	if len(podsToDelete) < 20 {
		return nil
	}

	existingPods := filterActivePods(r.Client, srcSet.GetNamespace(), podsToDelete)
	if reflect.DeepEqual(existingPods, podsToDelete) {
		return nil
	}

	// 更新 pod to delete 名单, 比较好理解
	// 对并发是否友好呢? 回忆之前和车正对于并发的讨论 compare and swap 已经 generation 问题
	utils.SetPodToDelete(srcSet, existingPods)
	if err := r.Update(context.TODO(), srcSet); err != nil {
		return fmt.Errorf("error truncate pods to delete")
	}

	return nil
}
