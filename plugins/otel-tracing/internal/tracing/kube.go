package tracing

import (
	"context"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterClient struct {
	Kubernetes kubernetes.Interface
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

	return &ClusterClient{Kubernetes: kube}, namespace, nil
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
