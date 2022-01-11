package utils

import (
	apps "k8s.io/api/apps/v1"
	"time"
)

type UpgradeStrategyType string

type UpgradeStrategy struct {
	Type           UpgradeStrategyType
	MaxUnavailable *int32
	Partition      *int32

	// the number of canary pods
	CanaryCount int64

	// timeout for one pod upgrading
	PodUpgradeTimeout time.Duration
}

const (
	UpgradingRolling UpgradeStrategyType = "RollingUpgrade"
)

func GetUpgradeStrategy(src *apps.StatefulSet) UpgradeStrategy {
	strategy := UpgradeStrategy{
		Type: UpgradingRolling,
	}

	if src.Spec.UpdateStrategy.RollingUpdate != nil {
		strategy.Partition = src.Spec.UpdateStrategy.RollingUpdate.Partition
	}

	//  annotation 中获取 maxUnavailble 以及 podUpgradeTimeout 参数
	// 不写了

	return strategy

}