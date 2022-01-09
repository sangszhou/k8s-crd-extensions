学习 kubebuilder v3 的 sample 项目

[1] 创建文件夹
```
mkdir k8s-crd-extensions
cd k8s-crd-extensions
```

[2] 创建脚手架 (只需要一次)

墙内网络, 需要先引入 proxy 才能下载到包

```
export GO111MODULE=on
export GOPROXY=https://goproxy.cn
```

创建脚手架, 保证目录是空的. 一个目录只需要执行一次脚手架

```
  
  kubebuilder init --domain tutorial.kubebuilder.io --repo tutorial.kubebuilder.io/project
```

创建完毕后, 只有 main.go 有一些内容, 没有其他的 go 代码

[3] 创建一个 api 和 controller

创建完毕后, 会生产 /api/v1 目录, cronjob_types.go 以及 controllers 目录
和 cronjob_controller.go 两个目录

```
kubebuilder create api --group batch --version v1 --kind CronJob
```

press `y` for resource and controller

CronJobType, CronJobList, CronJobSpec 和 CronJobStatus 都已经声明好了, 用户
只需要往里面填写数据即可.

