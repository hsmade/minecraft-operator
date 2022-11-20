module github.com/hsmade/minecraft-operator

go 1.16

require (
	github.com/go-logr/logr v1.1.0
	github.com/go-mc/mcping v1.2.1
	github.com/mitchellh/hashstructure/v2 v2.0.2
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/pkg/errors v0.9.1
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go/v11 v11.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.9.6
)
