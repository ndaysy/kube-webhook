package webhook

import (
    "k8s.io/api/admission/v1beta1"
    "net/http"
)

type IWebHookServer interface {
    //mutating(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse
    validating(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse
    Start()
    Stop()
}

// Webhook Server parameters
type WebHookServerParameters struct {
    Port           int          // webhook server port
    CertFile       string       // path to the x509 certificate for https
    KeyFile        string       // path to the x509 private key matching `CertFile`
    PortConfigFile string       // path to nodeport config file
    IgnoredNamespaces string    // ignored namespaces
}

type webHookServer struct {
    server *http.Server
    //sidecarConfig    *Config
}

//type Config struct {
//    Containers  []corev1.Container  `yaml:"containers"`
//    Volumes     []corev1.Volume     `yaml:"volumes"`
//}

//type patchOperation struct {
//    Op    string      `json:"op"`
//    Path  string      `json:"path"`
//    Value interface{} `json:"value,omitempty"`
//}

