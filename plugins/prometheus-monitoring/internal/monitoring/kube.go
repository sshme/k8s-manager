package monitoring

import (
	"context"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterClient struct {
	Kubernetes kubernetes.Interface
	APIExt     apiextensionsclient.Interface
	Dynamic    dynamic.Interface
	MonitorGVR schema.GroupVersionResource
}

func NewClusterClient(cfg Config) (*ClusterClient, string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if cfg.Kubeconfig != "" {
		loadingRules.ExplicitPath = cfg.Kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: cfg.Context}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	namespace := cfg.Namespace
	if namespace == "" {
		var overridden bool
		var err error
		namespace, overridden, err = clientConfig.Namespace()
		if err != nil {
			return nil, "", fmt.Errorf("resolve namespace: %w", err)
		}
		if !overridden && namespace == "" {
			namespace = "default"
		}
	}
	if namespace == "" {
		namespace = "default"
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, "", fmt.Errorf("load kubeconfig: %w", err)
	}

	kube, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, "", fmt.Errorf("create kubernetes client: %w", err)
	}
	apiExt, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		return nil, "", fmt.Errorf("create apiextensions client: %w", err)
	}
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, "", fmt.Errorf("create dynamic client: %w", err)
	}

	return &ClusterClient{
		Kubernetes: kube,
		APIExt:     apiExt,
		Dynamic:    dynamicClient,
		MonitorGVR: schema.GroupVersionResource{
			Group:    ServiceMonitorAPIGroup,
			Version:  ServiceMonitorAPIVersion,
			Resource: "servicemonitors",
		},
	}, namespace, nil
}

func ListDeploymentNames(ctx context.Context, client *ClusterClient, namespace string) ([]string, error) {
	list, err := client.Kubernetes.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments in namespace %q: %w", namespace, err)
	}
	names := make([]string, 0, len(list.Items))
	for _, deployment := range list.Items {
		names = append(names, deployment.Name)
	}
	sort.Strings(names)
	return names, nil
}

func GetDeployment(ctx context.Context, client *ClusterClient, namespace, name string) (*appsv1.Deployment, error) {
	deployment, err := client.Kubernetes.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get deployment %s/%s: %w", namespace, name, err)
	}
	return deployment, nil
}

func FindServiceForDeployment(ctx context.Context, client *ClusterClient, namespace string, deployment *appsv1.Deployment) (*corev1.Service, error) {
	services, err := client.Kubernetes.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list services in namespace %q: %w", namespace, err)
	}

	var matches []*corev1.Service
	for i := range services.Items {
		service := &services.Items[i]
		if serviceSelectsDeployment(*service, *deployment) {
			matches = append(matches, service)
		}
	}
	if len(matches) == 0 {
		return nil, nil
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name < matches[j].Name
	})
	return matches[0], nil
}

func HasPrometheusOperator(ctx context.Context, client *ClusterClient) (bool, error) {
	_, err := client.APIExt.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, ServiceMonitorCRDName, metav1.GetOptions{})
	if err == nil {
		return true, nil
	}
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	return false, fmt.Errorf("check ServiceMonitor CRD: %w", err)
}
