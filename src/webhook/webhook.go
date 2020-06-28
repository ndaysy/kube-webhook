package webhook

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/kubernetes/pkg/apis/core/v1"
	"kube-webhook/src/cache"
	"net/http"
	"strings"
	"sync"
)

var (
	once sync.Once
	ws   *webHookServer
	err  error
)

var (
	ignoredNamespaces = []string{
		metav1.NamespaceSystem,
		metav1.NamespacePublic,
	}
)

const (
	admissionWebhookAnnotationValidateKey = "aikube.com/validate"
	//admissionWebhookAnnotationMutateKey   = "aikube.com/mutate"
	//admissionWebhookAnnotationStatusKey   = "aikube.com/status"

	//nameLabel = "app.kubernetes.io/name"
	//
	//NA = "not_available"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
	defaulter     = runtime.ObjectDefaulter(runtimeScheme)
)

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	_ = v1.AddToScheme(runtimeScheme)
}

func NewWebhookServer(webHookParam WebHookServerParameters) (IWebHookServer, error) {
	once.Do(func() {
		if webHookParam.IgnoredNamespaces != "" {
			webHookParam.IgnoredNamespaces = strings.ReplaceAll(webHookParam.IgnoredNamespaces, " ", "")
			webHookParam.IgnoredNamespaces = strings.Trim(webHookParam.IgnoredNamespaces, ",")
			webHookParam.IgnoredNamespaces = strings.ToLower(webHookParam.IgnoredNamespaces)

			for _, ns := range strings.Split(webHookParam.IgnoredNamespaces, ",") {
				if ns != metav1.NamespaceSystem && ns != metav1.NamespacePublic {
					ignoredNamespaces = append(ignoredNamespaces, ns)
				}
			}
		}

		glog.Infof("Ignored namespaces:%v", ignoredNamespaces)

		ws, err = newWebHookServer(webHookParam)
	})
	return ws, err
}

func newWebHookServer(webHook WebHookServerParameters) (*webHookServer, error) {
	// load tls cert/key file
	tlsCertKey, err := tls.LoadX509KeyPair(webHook.CertFile, webHook.KeyFile)
	if err != nil {
		return nil, err
	}

	ws := &webHookServer{
		server: &http.Server{
			Addr:      fmt.Sprintf(":%v", webHook.Port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{tlsCertKey}},
		},
	}

	//sidecarConfig, err := loadConfig(webHook.SidecarCfgFile)
	if err != nil {
		return nil, err
	}

	// add routes
	mux := http.NewServeMux()
	//mux.HandleFunc("/mutating", ws.serve)
	mux.HandleFunc("/validating", ws.serve)
	ws.server.Handler = mux
	//ws.sidecarConfig = sidecarConfig
	return ws, nil
}

func (ws *webHookServer) Start() {
	if err := ws.server.ListenAndServeTLS("", ""); err != nil {
		glog.Errorf("Failed to listen and serve webhook server: %v", err)
	}
}

func (ws *webHookServer) Stop() {
	glog.Infof("Got OS shutdown signal, shutting down wenhook server gracefully...")
	ws.server.Shutdown(context.Background())
}

// validate deployments and services
func (whsvr *webHookServer) validating(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var (
		objectMeta                      *metav1.ObjectMeta
		resourceNamespace, resourceName string
	)

	glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, resourceName, req.UID, req.Operation, req.UserInfo)

	allowed := true
	result := &metav1.Status{}

	switch req.Kind.Kind {
	case "Service":
		var service corev1.Service
		if err := json.Unmarshal(req.Object.Raw, &service); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}

		resourceName, resourceNamespace, objectMeta = service.Name, service.Namespace, &service.ObjectMeta

		if !validationRequired(ignoredNamespaces, objectMeta) {
			glog.Infof("Skipping validation for %s/%s due to policy check", resourceNamespace, resourceName)
			return &v1beta1.AdmissionResponse{
				Allowed: true,
			}
		}

		// 只处理NodePort类型的service
		if service.Spec.Type == corev1.ServiceTypeNodePort && service.Spec.Ports != nil && len(service.Spec.Ports) > 0 {
			for _, servicePort := range service.Spec.Ports {
				//fmt.Println(service.Namespace, servicePort.NodePort)
				if !cache.PortCacheInstance().ExistKeyValue(int(servicePort.NodePort), service.Namespace) {
					glog.Infof("Unauthorized nodeport: rejected. namespace: %s, service: %s, nodeport: %d",resourceNamespace, resourceName, servicePort.NodePort )
					allowed = false
					result = &metav1.Status{
						Reason: "Unauthorized nodeport",
					}
					break
				}
				glog.Infof("Authorized nodeport: passed. namespace: %s, service: %s, nodeport: %d",resourceNamespace, resourceName, servicePort.NodePort )
			}
		}
	}

	return &v1beta1.AdmissionResponse{
		Allowed: allowed,
		Result:  result,
	}
}

// Serve method for webhook server
func (whsvr *webHookServer) serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		if r.URL.Path == "/validating" {
			admissionResponse = whsvr.validating(&ar)
		}
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

func admissionRequired(ignoredList []string, admissionAnnotationKey string, metadata *metav1.ObjectMeta) bool {
	// skip special kubernetes system namespaces
	for _, namespace := range ignoredList {
		if metadata.Namespace == namespace {
			glog.Infof("Skip validation for %v for it's in special namespace:%v", metadata.Name, metadata.Namespace)
			return false
		}
	}

	annotations := metadata.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	var required bool
	switch strings.ToLower(annotations[admissionAnnotationKey]) {
	default:
		required = true
	case "n", "no", "false", "off":
		required = false
	}
	return required
}

func validationRequired(ignoredList []string, metadata *metav1.ObjectMeta) bool {
	required := admissionRequired(ignoredList, admissionWebhookAnnotationValidateKey, metadata)
	glog.Infof("Validation policy for %v/%v: required:%v", metadata.Namespace, metadata.Name, required)
	return required
}
