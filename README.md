# 开发Kubernetes Operator

## Operator 介绍

Operator 可以看成是 CRD + Controller 的一种组合特例，Operator 是一种思想，它结合了特定领域知识并通过 CRD 机制扩展了 Kubernetes API 资源，使用户管理 Kubernetes 的内置资源（Pod、Deployment等）一样创建、配置和管理应用程序，Operator 是一个特定的应用程序的控制器，通过扩展 Kubernetes API 资源以代表 Kubernetes 用户创建、配置和管理复杂应用程序的实例，通常包含资源模型定义和控制器，通过 Operator 通常是为了实现某种特定软件（通常是有状态服务）的自动化运维。

我们完全可以通过上面的方式编写一个 CRD 对象，然后去手动实现一个对应的 Controller 就可以实现一个 Operator，但是我们也发现从头开始去构建一个 CRD 控制器并不容易，需要对 Kubernetes 的 API 有深入了解，并且 RBAC 集成、镜像构建、持续集成和部署等都需要很大工作量。为了解决这个问题，社区就推出了对应的简单易用的 Operator 框架，比较主流的是 kubebuilder 和 Operator Framework，这两个框架的使用基本上差别不大，我们可以根据自己习惯选择一个即可，我们这里先使用 Operator Framework 来给大家简要说明下 Operator 的开发。

## Operator Framework
Operator Framework 是 CoreOS 开源的一个用于快速开发 Operator 的工具包，该框架包含两个主要的部分：

Operator SDK: 无需了解复杂的 Kubernetes API 特性，即可让你根据你自己的专业知识构建一个 Operator 应用。
Operator Lifecycle Manager（OLM）: 帮助你安装、更新和管理跨集群的运行中的所有 Operator（以及他们的相关服务）
https://bxdc-static.oss-cn-beijing.aliyuncs.com/images/operator-sdk-lifecycle.png

Operator SDK 提供了用于开发 Go、Ansible 以及 Helm 中的 Operator 的工作流，下面的工作流适用于 Golang 的 Operator：

- 使用 SDK 创建一个新的 Operator 项目

- 通过添加自定义资源（CRD）定义新的资源 API

- 指定使用 SDK API 来 watch 的资源

- 定义 Operator 的协调（reconcile）逻辑

- 使用 Operator SDK 构建并生成 Operator 部署清单文件

每种 Operator 类型都有不同的功能集，在选择项目的类型时，重要的是要了解每种项目类型的功能和局限性以及 Operator 的用例。

https://bxdc-static.oss-cn-beijing.aliyuncs.com/images/20201016150157.png

## 示例

我们平时在部署一个简单的 Webserver 到 Kubernetes 集群中的时候，都需要先编写一个 Deployment 的控制器，然后创建一个 Service 对象，通过 Pod 的 label 标签进行关联，最后通过 Ingress 或者 type=NodePort 类型的 Service 来暴露服务，每次都需要这样操作，是不是略显麻烦，我们就可以创建一个自定义的资源对象，通过我们的 CRD 来描述我们要部署的应用信息，比如镜像、服务端口、环境变量等等，然后创建我们的自定义类型的资源对象的时候，通过控制器去创建对应的 Deployment 和 Service，是不是就方便很多了，相当于我们用一个资源清单去描述了 Deployment 和 Service 要做的两件事情。

这里我们将创建一个名为 AppService 的 CRD 资源对象，然后定义如下的资源清单进行应用部署：

```yaml
apiVersion: app.example.com/v1
kind: AppService
metadata:
  name: nginx-app
spec:
  size: 2
  image: nginx:1.7.9
  ports:
   - port: 80
     targetPort: 80
     nodePort: 30002
```



通过这里的自定义的 AppService 资源对象去创建副本数为2的 Pod，然后通过` nodePort=30002 `的端口去暴露服务，接下来我们就来一步一步的实现我们这里的这个简单的 Operator 应用。

## 开发环境
依赖服务版本：

- Kubernetes：v1.18.8
- operator-sdk：v1.17.0
- golang：1.17.3
- docker：20.10.12



安装Operator，其他服务安装略。

```shell
wget -c https://github.com/operator-framework/operator-sdk/releases/download/v1.17.0/operator-sdk_linux_amd64

mv operator-sdk_linux_amd64 /usr/bin/operator-sdk
chmod +x /usr/bin/operator-sdk
```

还需要安装gcc

```shell
yum -y install gcc
```

## Operator开发

环境准备好了，接下来就可以使用 operator-sdk 直接创建一个新的项目了，命令格式为：`operator-sdk init`。

按照上面我们预先定义的 CRD 资源清单，我们这里可以这样创建：

```shell
# 在GOPATH目录下操作(非必须)
mkdir -p my-operator && cd my-operator
# 使用gomodules包管理工具
export GO111MODULE=on  
# 使用代理加速
export GOPROXY="https://goproxy.cn" 
# 使用 sdk 创建一个名为 opdemo 的 operator 项目，如果在GOPATH之外需要指定 repo 参数
go mod init my-operator

# 使用下面的命令初始化项目
operator-sdk init --domain example.com --license apache2 --owner "developer"
```


初始化完成后的项目结构如下所示：

```shell
$ tree -L 2
.
├── config
│   ├── default
│   ├── manager
│   ├── manifests
│   ├── prometheus
│   ├── rbac
│   └── scorecard
├── Dockerfile
├── go.mod
├── go.sum
├── hack
│   └── boilerplate.go.txt
├── main.go
├── Makefile
└── PROJECT

8 directories, 7 files
```

到这里一个全新的 Operator 项目就新建完成了。

### 项目结构

使用 operator-sdk init 命令创建新的 Operator 项目后，项目目录就包含了很多生成的文件夹和文件。

- go.mod/go.sum  - Go Modules 包管理清单，用来描述当前 Operator 的依赖包。
- main.go 文件，使用 operator-sdk API 初始化和启动当前 Operator 的入口。
- deploy - 包含一组用于在 Kubernetes 集群上进行部署的通用的 Kubernetes 资源清单文件。
- pkg/apis - 包含定义的 API 和自定义资源（CRD）的目录树，这些文件允许 sdk 为 CRD 生成代码并注册对应的类型，以便正确解码自定义资源对象。
- pkg/controller - 用于编写所有的操作业务逻辑的地方
- version - 版本定义
- build - Dockerfile 定义目录

我们主要需要编写的是 pkg 目录下面的 api 定义以及对应的 controller 实现。

### 添加 API

接下来为我们的自定义资源添加一个新的 API，按照上面我们预定义的资源清单文件，在 Operator 相关根目录下面执行如下命令：

```shell
$ operator-sdk create api --resource=true --controller=true --group app --version v1 --kind AppService
$ go mod tidy
```

这里我们添加了一个 group 为 app，版本为 v1beta1 的 AppService 的资源对象，添加完成后，我们可以看到类似于下面的这样项目结构，我们可以看到生成了对应的 api 和 controllers 包：

https://bxdc-static.oss-cn-beijing.aliyuncs.com/images/20201016161654.png

### 自定义 API

打开源文件` api/v1beta1/appservice_types.go`，需要我们根据我们的需求去自定义结构体 AppServiceSpec，我们最上面预定义的资源清单中就有 size、image、ports 这些属性，所有我们需要用到的属性都需要在这个结构体中进行定义：


```go
type AppServiceSpec struct {
	Size      *int32                        `json:"size"`
	Image     string                        `json:"image"`
	Resources corev1.ResourceRequirements   `json:"resources,omitempty"`
	Envs      []corev1.EnvVar               `json:"envs,omitempty"`
	Ports     []corev1.ServicePort          `json:"ports,omitempty"`
}
```

代码中会涉及到一些包名的导入，由于包名较多，所以我们会使用一些别名进行区分，主要的包含下面几个：

```go
import (
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)
```

这里的 resources、envs、ports 的定义都是直接引用的 "k8s.io/api/core/v1" 中定义的结构体，而且需要注意的是我们这里使用的是 ServicePort，而不是像传统的 Pod 中定义的 ContanerPort，这是因为我们的资源清单中不仅要描述容器的 Port，还要描述 Service 的 Port。

然后一个比较重要的结构体 AppServiceStatus 用来描述资源的状态，当然我们可以根据需要去自定义状态的描述，我这里就偷懒直接使用 Deployment 的状态了：

```go
type AppServiceStatus struct {
	appsv1.DeploymentStatus `json:",inline"`
}
```

定义完成后，在项目根目录下面执行如下命令：

```shell
$ make
/Users/ych/devs/projects/go/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
go fmt ./...
go vet ./...
go build -o bin/manager main.go
```

该命令会使用我们更新后的资源对象结构重新自动生成一些代码，这样我们就算完成了对自定义资源对象的 API 的声明。

### 实现业务逻辑

上面 API 描述声明完成了，接下来就需要我们来进行具体的业务逻辑实现了，编写具体的 controller 实现，打开源文件`controllers/appservice_controller.go`，需要我们去更改的地方也不是很多，核心的就是Reconcile 方法，该方法就是去不断的 watch 资源的状态，然后根据状态的不同去实现各种操作逻辑。

首先 sdk 为我们搭建了一个基本的 reconciler 结构，几乎每一个调谐器器都需要记录日志，并且能够获取对象，所以可以直接使用。

```go
type AppServiceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}
```

在appservice_controller.go文件中定义一个全局变量，用于后面的annotations

```go
var oldSpecAnnotation = "old/spec"
```

Reconcile 实际上是对单个对象进行调谐，我们的 Request 只是有一个名字，但我们可以使用 client 从缓存中获取这个对象。我们返回一个空的结果，没有错误，这就向 controller-runtime 表明我们已经成功地对这个对象进行了调谐，在有一些变化之前不需要再尝试调谐。

大多数控制器需要一个日志句柄和一个上下文，所以我们在 Reconcile 中将他们初始化。上下文是用来允许取消请求的，它是所有 client 方法的第一个参数。

Reconcile 业务逻辑代码如下。

```go
func (r *AppServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
ctx := context.Background()
	log := r.Log.WithValues("appservice", req.NamespacedName)

	// 业务逻辑实现
	// 获取 AppService 实例
	var appService appv1beta1.AppService
	err := r.Get(ctx, req.NamespacedName, &appService)
	if err != nil {
		// MyApp 被删除的时候，忽略
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if appService.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}
	log.Info("fetch appservice objects", "appservice", appService)

	// CreateOrUpdate Deployment
	var deploy appsv1.Deployment
	deploy.Name = appService.Name
	deploy.Namespace = appService.Namespace
	or, err := ctrl.CreateOrUpdate(ctx, r, &deploy, func() error {
		MutateDeployment(&appService, &deploy)
		return controllerutil.SetControllerReference(&appService, &deploy, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("CreateOrUpdate", "Deployment", or)

	// CreateOrUpdate Service
	var service v1.Service
	service.Name = appService.Name
	service.Namespace = appService.Namespace
	or, err = ctrl.CreateOrUpdate(ctx, r, &service, func() error {
		MutateService(&appService, &service)
		return controllerutil.SetControllerReference(&appService, &service, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("CreateOrUpdate", "Service", or)
	return ctrl.Result{}, nil
```
上面就是业务逻辑实现的核心代码，逻辑很简单，就是去判断资源是否存在，不存在，则直接创建新的资源，创建新的资源除了需要创建 Deployment 资源外，还需要创建 Service 资源对象，因为这就是我们的需求，当然你还可以自己去扩展，比如再创建一个 Ingress 对象。更新也是一样的，去对比新旧对象的声明是否一致，如果不一致则需要更新，同样的，两种资源都需要更新的。

另外两个核心的方法就是上面的 resources.NewDeploy(instance) 和 resources.NewService(instance) 方法，这两个方法实现逻辑也很简单，就是根据 CRD 中的声明去填充 Deployment 和 Service 资源对象的 Spec 对象即可。

在groupversion_info.go文件里增加一个Kind全局变量

```go
var (
	GroupVersion = schema.GroupVersion{Group: "app.ydzs.io", Version: "v1beta1"}
	Kind         = "AppService"
    SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
    AddToScheme = SchemeBuilder.AddToScheme
)
```

NewDeploy 方法实现如下：

```

```


newService 对应的方法实现如下：

```

```



这样我们就实现了 AppService 这种资源对象的业务逻辑。

### 调试

如果我们本地有一个可以访问的 Kubernetes 集群，我们也可以直接进行调试，在本地用户 ~/.kube/config 文件中配置集群访问信息，下面的信息表明可以访问 Kubernetes 集群：

```shell
$ kubectl cluster-info
Kubernetes master is running at https://ydzs-master:6443
KubeDNS is running at https://ydzs-master:6443/api/v1/namespaces/kube-system/services/kube-dns/proxy
To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
```

首先，需要在集群中安装 CRD 对象：

```shell
$ make install 

$ kubectl get crd |grep appservice
appservices.app.example.com             2022-02-27T07:31:10Z
```

当我们通过 kubectl get crd 命令获取到我们定义的 CRD 资源对象，就证明我们定义的 CRD 安装成功了。其实现在只是 CRD 的这个声明安装成功了，但是我们这个 CRD 的具体业务逻辑实现方式还在我们本地，并没有部署到集群之中，我们可以通过下面的命令来在本地项目中启动 Operator 的调试：

```shell
$ make run ENABLE_WEBHOOKS=false
```

上面的命令会在本地运行 Operator 应用，通过 ~/.kube/config 去关联集群信息，现在我们去添加一个 AppService 类型的资源然后观察本地 Operator 的变化情况，资源清单文件就是我们上面预定义的（config/samples/app_v1beta1_appservice.yaml）:

```yaml
apiVersion: app.ydzs.io/v1beta1
kind: AppService
metadata:
  name: nginx
spec:
  size: 2
  image: nginx:1.7.9
  ports:
   - port: 80
     targetPort: 80
     nodePort: 30002
```




直接创建这个资源对象：

```shell
$ kubectl apply -f config/samples/app_v1beta1_appservice.yaml
appservice.app.ydzs.io/nginx-app created
```

我们可以看到我们的应用创建成功了，这个时候查看 Operator 的调试窗口会有如下的信息出现：

```shell
......
{"level":"info","ts":1559207416.670523,"logger":"controller_appservice","msg":"Reconciling AppService","Request.Namespace":"default","Request.Name":"nginx-app"}
{"level":"info","ts":1559207417.004226,"logger":"controller_appservice","msg":"Reconciling AppService","Request.Namespace":"default","Request.Name":"nginx-app"}
{"level":"info","ts":1559207417.004331,"logger":"controller_appservice","msg":"Reconciling AppService","Request.Namespace":"default","Request.Name":"nginx-app"}
{"level":"info","ts":1559207418.33779,"logger":"controller_appservice","msg":"Reconciling AppService","Request.Namespace":"default","Request.Name":"nginx-app"}
{"level":"info","ts":1559207418.951193,"logger":"controller_appservice","msg":"Reconciling AppService","Request.Namespace":"default","Request.Name":"nginx-app"}
......
```

然后我们可以去查看集群中是否有符合我们预期的资源出现：

```shell
$ kubectl get AppService
NAME        AGE
nginx       2m8s
$ kubectl get deploy
NAME                     READY   UP-TO-DATE   AVAILABLE   AGE
nginx   p                2/2     2            2           2m20s
$ kubectl get svc   
NAME             TYPE           CLUSTER-IP       EXTERNAL-IP             PORT(S)          AGE
nginx            NodePort       10.111.179.0     <none>                  80:30002/TCP     2m23s
```



看到了吧，我们定义了两个副本（size=2），这里就出现了两个 Pod，还有一个 NodePort=30002 的 Service 对象，我们可以通过该端口去访问下应用：

https://bxdc-static.oss-cn-beijing.aliyuncs.com/images/operator-demo-op-demo.png

如果应用在安装过程中出现了任何问题，我们都可以通过本地的 Operator 调试窗口找到有用的信息，然后调试修改即可。

清理：

```shell
$ kubectl delete -f config/samples/app_v1beta1_appservice.yaml
$ make uninstall
```


## 部署

自定义的资源对象现在测试通过了，但是如果我们将本地的调试控制器终止掉，我们可以猜想到就没办法处理 AppService 资源对象的一些操作了，所以我们需要将我们的业务逻辑实现部署到集群中去。

执行下面的命令构建 Operator 应用打包成 Docker 镜像：

Dockerfile 需要更改为这样：

```dockerfile
# Build the manager binary
FROM golang:1.13 as builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN export GOPROXY=https://goproxy.cn && go mod download   # 增加代理

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

FROM golang:1.13  # 将distrolers镜像改掉，该镜像用于减小镜像大小的
WORKDIR /
COPY --from=builder /workspace/manager .

ENTRYPOINT ["/manager"]
```



执行命令

```shell
$ export USERNAME=<dockerbub-username>
$ make docker-build IMG=$USERNAME/opdemo:v1.0.0
......
Successfully built 29cd605c4ad2
Successfully tagged cnych/opdemo:v1.0.0
INFO[0041] Operator build complete. 
```

镜像构建成功后，推送到 docker hub：

```shell
$ make docker-push IMG=$USERNAME/opdemo:v1.0.0
```

将manager.yaml里的镜像名改为你打包后的镜像名

```yaml
apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: system
  labels:
    control-plane: controller-manager
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  replicas: 1
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - command:
        - /manager
        args:
        - --enable-leader-election
        image: sfeng/opdemo:v1.0.1  // 此处
        name: manager
        resources:
          limits:
            cpu: 100m
            memory: 30Mi
          requests:
            cpu: 100m
            memory: 20Mi
        imagePullPolicy: IfNotPresent
      terminationGracePeriodSeconds: 10
```

提前下载kube-rbac镜像，该镜像在境外。

使用国内 kubesphere 公司提供的，然后下载就可以了。（这个可以到 docker-hub 中搜索，然后找出国内谁家有提供） 之前本来国内可以使用 gcr.azk8s.cn 来下载 gcr.io 的镜像，但现在禁止了，只有 aws 的 IP 才可以。

```shell
$ docker pull kubesphere/kube-rbac-proxy:v0.5.0
$ docker tag kubesphere/kube-rbac-proxy:v0.5.0 gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0
$ make deploy IMG=$USERNAME/opdemo:v1.0.0
```

现在 Operator 的资源清单文件准备好了，然后就可以使用下面的命令来部署 CRD 资源对象了：

```shell
$ kubectl apply -f config/samples/app_v1beta1_appservice.yaml
$ kubectl get crd |grep myapp
myapps.app.ydzs.io   2020-11-06T07:06:54Z
```

到这里我们的 CRD 和 Operator 实现都已经安装成功了。
