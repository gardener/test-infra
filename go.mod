module github.com/gardener/test-infra

go 1.12

require (
	cloud.google.com/go v0.40.0
	github.com/Masterminds/semver v1.4.2
	github.com/Masterminds/sprig v2.17.1+incompatible // indirect
	github.com/aokoli/goutils v1.1.0 // indirect
	github.com/argoproj/argo v2.3.0+incompatible
	github.com/cpuguy83/go-md2man v1.0.8 // indirect
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/emicklei/go-restful v2.8.1+incompatible // indirect
	github.com/gardener/controller-manager-library v0.0.0-20190531111244-4db8db4aed9b // indirect
	github.com/gardener/external-dns-management v0.0.0-20190523072504-715c7cc74e89 // indirect
	github.com/gardener/gardener v0.0.0-20190628060349-06358ff44e46
	github.com/gardener/gardener-extensions v0.0.0-20190620142130-227adb277fa4 // indirect
	github.com/gardener/gardener-resource-manager v0.0.0-20190627140746-b43a76cdf9a7 // indirect
	github.com/gardener/machine-controller-manager v0.0.0-20190228095106-36a42c48af0a // indirect
	github.com/ghodss/yaml v0.0.0-20190212211648-25d852aebe32 // indirect
	github.com/go-ini/ini v1.41.0 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.0
	github.com/go-openapi/jsonpointer v0.18.0 // indirect
	github.com/go-openapi/jsonreference v0.18.0 // indirect
	github.com/go-openapi/spec v0.18.0
	github.com/go-openapi/swag v0.18.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.2.0 // indirect
	github.com/golang/groupcache v0.0.0-20181024230925-c65c006176ff // indirect
	github.com/google/go-github/v27 v27.0.4
	github.com/google/uuid v1.1.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v0.0.0-20180717150148-3d5d8f294aa0
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/minio/minio-go v6.0.13+incompatible
	github.com/mitchellh/go-homedir v1.0.0 // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/olekukonko/tablewriter v0.0.1
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pborman/uuid v0.0.0-20180906182336-adf5a7427709 // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.2 // indirect
	github.com/prometheus/client_model v0.0.0-20190115171406-56726106282f // indirect
	github.com/prometheus/common v0.1.0 // indirect
	github.com/prometheus/procfs v0.0.0-20190117184657-bf6a532e95b1 // indirect
	github.com/russross/blackfriday v1.5.2 // indirect
	github.com/sirupsen/logrus v1.3.0
	github.com/smartystreets/goconvey v0.0.0-20190731233626-505e41936337 // indirect
	github.com/spf13/cobra v0.0.0-20181021141114-fe5e611709b0
	github.com/spf13/pflag v1.0.3 // indirect
	go.opencensus.io v0.22.0 // indirect
	go.uber.org/zap v1.9.1
	golang.org/x/crypto v0.0.0-20190325154230-a5d413f7728c // indirect
	golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	google.golang.org/api v0.6.0
	google.golang.org/genproto v0.0.0-20190611190212-a7e196e89fd3
	google.golang.org/grpc v1.21.1 // indirect
	gopkg.in/ini.v1 v1.44.0 // indirect
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apiextensions-apiserver v0.0.0-20190409022649-727a075fdec8
	k8s.io/apimachinery v0.0.0-20190612125636-6a5db36e93ad
	k8s.io/apiserver v0.0.0-20190313205120-8b27c41bdbb1 // indirect
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.0.0-00010101000000-000000000000
	k8s.io/component-base v0.0.0-20190617074208-2b0aae80ca81 // indirect
	k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a // indirect
	k8s.io/helm v2.12.2+incompatible
	k8s.io/klog v0.3.1
	k8s.io/kube-aggregator v0.0.0-20190116053718-60c339211c1a // indirect
	k8s.io/kube-openapi v0.0.0-20190228160746-b3a7cee44a30
	k8s.io/utils v0.0.0-20190529001817-6999998975a7 // indirect
	sigs.k8s.io/controller-runtime v0.2.0-beta.2
	sigs.k8s.io/yaml v1.1.0
)

replace (
	github.com/onsi/ginkgo => github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega => github.com/onsi/gomega v1.5.0
	k8s.io/api => k8s.io/api v0.0.0-20190313235455-40a48860b5ab //kubernetes-1.14.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190315093550-53c4693659ed // kubernetes-1.14.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190313205120-d7deff9243b1 // kubernetes-1.14.0
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190313205120-8b27c41bdbb1 // kubernetes-1.14.0
	k8s.io/client-go => k8s.io/client-go v11.0.0+incompatible // kubernetes-1.14.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190314002537-50662da99b70 // kubernetes-1.14.0
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190311093542-50b561225d70 // kubernetes-1.14.0
	k8s.io/component-base => k8s.io/component-base v0.0.0-20190314000054-4a91899592f4 // kubernetes-1.14.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190314000639-da8327669ac5 // kubernetes-1.14.0
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190320154901-5e45bb682580
)
