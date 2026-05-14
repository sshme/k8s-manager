package k8s

import (
	"fmt"
	"sort"

	"k8s.io/client-go/tools/clientcmd"
)

type ContextInfo struct {
	Name             string
	Cluster          string
	DefaultNamespace string
}

// LoadContexts считывает объединенный kubeconfig (учитывая переменную $KUBECONFIG)
// и возвращает список доступных контекстов, а также имя текущего контекста.
// Не выполняет сетевых запросов.
func LoadContexts() ([]ContextInfo, string, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	raw, err := rules.Load()
	if err != nil {
		return nil, "", fmt.Errorf("load kubeconfig: %w", err)
	}
	if raw == nil {
		return nil, "", nil
	}

	contexts := make([]ContextInfo, 0, len(raw.Contexts))
	for name, ctx := range raw.Contexts {
		if ctx == nil {
			continue
		}
		contexts = append(contexts, ContextInfo{
			Name:             name,
			Cluster:          ctx.Cluster,
			DefaultNamespace: ctx.Namespace,
		})
	}
	sort.Slice(contexts, func(i, j int) bool {
		return contexts[i].Name < contexts[j].Name
	})
	return contexts, raw.CurrentContext, nil
}
