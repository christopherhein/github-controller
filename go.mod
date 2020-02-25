module go.hein.dev/github-controller

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-github/v28 v28.1.1
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	sigs.k8s.io/controller-runtime v0.4.0
)

replace sigs.k8s.io/controller-runtime => go.hein.dev/controller-runtime v0.4.1-0.20200302183002-75376c7e0696
