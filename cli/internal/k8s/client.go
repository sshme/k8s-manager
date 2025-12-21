package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

func NewClient() (*Client, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home dir: %w", err)
			}
			kubeconfig = filepath.Join(home, ".kube", "config")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &Client{
		clientset: clientset,
		config:    config,
	}, nil
}

func (c *Client) ApplyManifest(ctx context.Context, yamlData []byte) error {
	decoder := kyaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlData), 100)
	
	dynamicClient, err := dynamic.NewForConfig(c.config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode YAML: %w", err)
		}

		if len(rawObj.Raw) == 0 {
			continue
		}

		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to decode object: %w", err)
		}

		unstructuredObj, ok := obj.(*unstructured.Unstructured)
		if !ok {
			return fmt.Errorf("object is not unstructured")
		}

		// Get GVR from GVK
		gvr := schema.GroupVersionResource{
			Group:    gvk.Group,
			Version:  gvk.Version,
			Resource: getResourceFromKind(gvk.Kind),
		}

		// Get resource interface
		resourceInterface := dynamicClient.Resource(gvr)
		ns := unstructuredObj.GetNamespace()
		if ns != "" {
			resourceInterface = resourceInterface.Namespace(ns)
		}

		// Use server-side apply
		unstructuredObj.SetManagedFields(nil) // Clear managed fields for apply
		_, err = resourceInterface.Apply(ctx, unstructuredObj.GetName(), unstructuredObj, v1.ApplyOptions{
			FieldManager: "k8s-manager",
		})
		if err != nil {
			return fmt.Errorf("failed to apply resource %s/%s: %w", gvk.Kind, unstructuredObj.GetName(), err)
		}
	}

	return nil
}

func getGroupVersionResource(obj *unstructured.Unstructured) (schema.GroupVersionResource, error) {
	gvk := obj.GroupVersionKind()
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: getResourceFromKind(gvk.Kind),
	}, nil
}

func getResourceFromKind(kind string) string {
	// Simple mapping - in production use discovery API
	kindToResource := map[string]string{
		"Deployment":  "deployments",
		"Service":     "services",
		"ConfigMap":   "configmaps",
		"Secret":      "secrets",
		"Ingress":     "ingresses",
		"Namespace":   "namespaces",
		"DaemonSet":   "daemonsets",
		"StatefulSet": "statefulsets",
	}
	
	if resource, ok := kindToResource[kind]; ok {
		return resource
	}
	return fmt.Sprintf("%ss", strings.ToLower(kind))
}

