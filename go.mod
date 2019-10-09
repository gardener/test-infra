module github.com/gardener/test-infra

go 1.13

require (
	cloud.google.com/go v0.43.0
	github.com/Masterminds/semver v1.4.2
	github.com/argoproj/argo v2.4.0+incompatible
	github.com/bradleyfalzon/ghinstallation v0.1.2
	github.com/elazarl/goproxy v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/gardener/gardener v0.0.0-20190906111529-f9ad04069615
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1
	github.com/go-openapi/spec v0.19.2
	github.com/golang/mock v1.3.1
	github.com/google/go-github/v27 v27.0.4
	github.com/gorilla/mux v1.6.2
	github.com/hashicorp/go-multierror v1.0.0
	github.com/joho/godotenv v1.3.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/minio/minio-go v6.0.13+incompatible
	github.com/olekukonko/tablewriter v0.0.1
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/smartystreets/goconvey v0.0.0-20190731233626-505e41936337 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.4.0
	go.uber.org/zap v1.10.0
	golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	google.golang.org/api v0.7.0
	google.golang.org/genproto v0.0.0-20190801165951-fa694d86fc64
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apiextensions-apiserver v0.0.0-20190409022649-727a075fdec8
	k8s.io/apimachinery v0.0.0-20190612125636-6a5db36e93ad
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.0.0-20190713022532-93d7507fc8ff
	k8s.io/component-base v0.0.0-20190617074208-2b0aae80ca81 // indirect
	k8s.io/helm v2.14.2+incompatible
	k8s.io/kube-openapi v0.0.0-20190722073852-5e22f3d471e6
	sigs.k8s.io/controller-runtime v0.2.0-beta.2
	sigs.k8s.io/yaml v1.1.0
)

replace (
	github.com/census-instrumentation/opencensus-proto => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	github.com/onsi/ginkgo => github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega => github.com/onsi/gomega v1.5.0
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
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
