apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: validation-kube-webhook-cfg
  namespace: ikube
  labels:
    app: admission-kube-webhook
webhooks:
  - name: nodeport.kube-webhook.cn
    clientConfig:
      service:
        name: admission-kube-webhook-svc
        namespace: ikube
        path: "/validating"
      caBundle: ${CA_BUNDLE}
    rules:
      - operations: [ "CREATE", "UPDATE" ]
        apiGroups: ["apps", "extensions", ""]
        apiVersions: ["v1", "v1beta1"]
        resources: ["services"]
    namespaceSelector:
      matchLabels:
        admission-kube-webhook: enabled
 #   namespaceSelector: {}
