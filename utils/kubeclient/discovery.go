package kubeclient

import "k8s.io/apimachinery/pkg/runtime/schema"

/**
查看 k8s 是否安装了对应的资源
*/
func IsResourceInstalled(gvk schema.GroupVersionKind) (bool, error) {
	resourceList, err := discoveryClient.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return false, err
	}

	for _, rs := range resourceList.APIResources {
		if rs.Kind == gvk.Kind {
			return true, nil
		}
	}

	return false, nil

}
