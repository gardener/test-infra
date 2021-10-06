module github.com/gardener/test-infra

go 1.16

require (
	cloud.google.com/go v0.81.0
	cloud.google.com/go/storage v1.10.0
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/Microsoft/hcsshim v0.8.14 // indirect
	github.com/argoproj/argo/v2 v2.12.10
	github.com/bradleyfalzon/ghinstallation v1.1.1
	github.com/containerd/containerd v1.4.3 // indirect
	github.com/containerd/continuity v0.0.0-20201208142359-180525291bb7 // indirect
	github.com/docker/cli v20.10.3+incompatible // indirect
	github.com/emicklei/go-restful v2.11.1+incompatible // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gardener/component-cli v0.28.0
	github.com/gardener/component-spec/bindings-go v0.0.52
	github.com/gardener/gardener v1.19.1
	github.com/ghodss/yaml v1.0.0
	github.com/go-ini/ini v1.51.1 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.3.0
	github.com/go-openapi/spec v0.19.8
	github.com/golang/mock v1.5.0
	github.com/google/go-github/v27 v27.0.4
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/sessions v1.1.3
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/hashicorp/go-multierror v1.0.0
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/mandelsoft/vfs v0.0.0-20210530103237-5249dc39ce91
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.2.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602
	google.golang.org/api v0.44.0
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.20.2
	k8s.io/helm v2.16.1+incompatible
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	k8s.io/metrics v0.20.2
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// helm dependencies
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/onsi/ginkgo => github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega => github.com/onsi/gomega v1.5.0
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
	k8s.io/api => k8s.io/api v0.19.9
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.9
	k8s.io/client-go => k8s.io/client-go v0.19.9
	k8s.io/code-generator => k8s.io/code-generator v0.19.9
	k8s.io/helm => k8s.io/helm v2.13.1+incompatible
)
