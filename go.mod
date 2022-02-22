module github.com/gardener/test-infra

go 1.16

require (
	cloud.google.com/go v0.81.0
	cloud.google.com/go/storage v1.10.0
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/argoproj/argo-workflows/v3 v3.1.8
	github.com/bradleyfalzon/ghinstallation/v2 v2.0.3
	github.com/docker/cli v20.10.3+incompatible // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gardener/component-cli v0.32.0
	github.com/gardener/component-spec/bindings-go v0.0.53
	github.com/gardener/gardener v1.38.2
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-ini/ini v1.51.1 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0
	github.com/go-openapi/spec v0.20.2
	github.com/golang-jwt/jwt/v4 v4.1.0 // indirect
	github.com/golang/mock v1.6.0
	github.com/google/go-github/v39 v39.2.0
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/sessions v1.1.3
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/hashicorp/go-multierror v1.1.0
	github.com/joho/godotenv v1.3.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/mandelsoft/vfs v0.0.0-20210530103237-5249dc39ce91
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	go.uber.org/zap v1.19.0
	golang.org/x/crypto v0.0.0-20211117183948-ae814b36b871 // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602
	google.golang.org/api v0.44.0
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.2
	k8s.io/apiextensions-apiserver v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.22.2
	k8s.io/helm v2.16.1+incompatible
	k8s.io/kube-openapi v0.0.0-20210421082810-95288971da7e
	k8s.io/metrics v0.22.2
	k8s.io/utils v0.0.0-20210819203725-bdf08cb9a70a
	sigs.k8s.io/controller-runtime v0.10.2
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// helm dependencies
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v20.10.11+incompatible
	github.com/onsi/ginkgo => github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega => github.com/onsi/gomega v1.5.0
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
	k8s.io/api => k8s.io/api v0.19.9
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.9
	k8s.io/client-go => k8s.io/client-go v0.19.9
	k8s.io/code-generator => k8s.io/code-generator v0.19.9
	k8s.io/helm => k8s.io/helm v2.17.0+incompatible
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	// needed because of https://github.com/kubernetes-sigs/controller-runtime/issues/1538
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.8.3
)
