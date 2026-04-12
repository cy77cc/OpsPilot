// Package logic 提供部署相关的业务逻辑实现。
//
// 本文件实现将 YAML 配置部署到 Kubernetes 集群的核心功能。
package logic

import (
	"context"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/model"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// DeployToCluster 将 YAML 内容部署到指定 Kubernetes 集群。
//
// 参数:
//   - ctx: 上下文
//   - cluster: 目标集群模型，包含 KubeConfig
//   - yamlContent: YAML 配置内容，支持多文档 (--- 分隔)
//
// 返回: 成功返回 nil，失败返回错误
//
// 实现细节:
//   - 使用 dynamic client 进行通用资源操作
//   - 支持多 YAML 文档部署
//   - 使用 Server-Side Apply 确保声明式管理
//   - 自动处理命名空间资源 (默认 default)
func DeployToCluster(ctx context.Context, cluster *model.Cluster, yamlContent string) error {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))

	// Split YAML documents
	docs := strings.Split(yamlContent, "---")

	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	for _, doc := range docs {
		if strings.TrimSpace(doc) == "" {
			continue
		}

		obj := &unstructured.Unstructured{}
		_, gvk, err := dec.Decode([]byte(doc), nil, obj)
		if err != nil {
			return err
		}

		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			namespace := obj.GetNamespace()
			if namespace == "" {
				namespace = "default"
			}
			dr = dynamicClient.Resource(mapping.Resource).Namespace(namespace)
		} else {
			dr = dynamicClient.Resource(mapping.Resource)
		}

		// Server-Side Apply
		data, err := obj.MarshalJSON()
		if err != nil {
			return err
		}

		// Force set apiVersion and kind in the data to ensure they are present for Apply
		// (Unstructured MarshalJSON should include them)

		_, err = dr.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: "OpsPilot",
		})
		if err != nil {
			return err
		}
	}
	return nil
}
