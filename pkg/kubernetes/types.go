package kubernetes

import (
	openapi_v2 "github.com/google/gnostic/openapiv2"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	Client        kubernetes.Interface
	RestClient    rest.Interface
	Config        *rest.Config
	ServerVersion *version.Info
	CtrlClient    ctrl.Client
}

type K8sApiReference struct {
	ApiVersion    schema.GroupVersion
	Kind          string
	OpenapiSchema *openapi_v2.Document
}
