// Package logic 提供服务管理业务逻辑实现。
//
// 本文件实现服务的创建、更新、部署和 K8s YAML 生成等核心业务逻辑。
package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	v1 "github.com/cy77cc/OpsPilot/api/project/v1"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
)

// ServiceLogic 是服务管理的业务逻辑层。
//
// 职责:
//   - 处理服务的 CRUD 操作
//   - 生成 K8s Deployment/StatefulSet 和 Service YAML
//   - 执行服务部署到指定集群
type ServiceLogic struct {
	svcCtx *svc.ServiceContext
}

// NewServiceLogic 创建服务逻辑层实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接等依赖
//
// 返回: 服务逻辑层实例
func NewServiceLogic(svcCtx *svc.ServiceContext) *ServiceLogic {
	return &ServiceLogic{
		svcCtx: svcCtx,
	}
}

// CreateService 创建服务。
//
// 参数:
//   - ctx: 上下文
//   - req: 创建请求，包含服务配置信息
//
// 返回: 创建的服务信息，失败返回错误
//
// 流程:
//  1. 生成或使用提供的 K8s YAML
//  2. 序列化环境变量和资源配置
//  3. 持久化到数据库
func (l *ServiceLogic) CreateService(ctx context.Context, req v1.CreateServiceReq) (v1.ServiceResp, error) {
	// 1. Generate or Use K8s YAML
	var yamlContent string
	var err error

	if req.YamlContent != "" {
		yamlContent = req.YamlContent
	} else {
		yamlContent, err = l.generateK8sYAML(req)
		if err != nil {
			return v1.ServiceResp{}, fmt.Errorf("failed to generate YAML: %v", err)
		}
	}

	// 2. Marshal JSON fields
	var envVarsBytes, resourcesBytes []byte
	if req.EnvVars != nil {
		envVarsBytes, _ = json.Marshal(req.EnvVars)
	}
	if req.Resources != nil {
		resourcesBytes, _ = json.Marshal(req.Resources)
	}

	service := &model.Service{
		ProjectID:     req.ProjectID,
		Name:          req.Name,
		Type:          req.Type,
		Image:         req.Image,
		Replicas:      req.Replicas,
		ServicePort:   req.ServicePort,
		ContainerPort: req.ContainerPort,
		NodePort:      req.NodePort,
		EnvVars:       string(envVarsBytes),
		Resources:     string(resourcesBytes),
		YamlContent:   yamlContent,
	}

	if err := l.svcCtx.DB.Create(service).Error; err != nil {
		return v1.ServiceResp{}, err
	}

	return l.modelToResp(service), nil
}

// ListServices 获取服务列表。
//
// 参数:
//   - ctx: 上下文
//   - projectID: 项目 ID，为 0 时查询所有服务
//
// 返回: 服务列表，失败返回错误
func (l *ServiceLogic) ListServices(ctx context.Context, projectID uint) ([]v1.ServiceResp, error) {
	var services []model.Service
	query := l.svcCtx.DB
	if projectID > 0 {
		query = query.Where("project_id = ?", projectID)
	}
	if err := query.Find(&services).Error; err != nil {
		return nil, err
	}

	var res []v1.ServiceResp
	for _, s := range services {
		res = append(res, l.modelToResp(&s))
	}
	return res, nil
}

// GetService 获取服务详情。
//
// 参数:
//   - ctx: 上下文
//   - id: 服务 ID
//
// 返回: 服务详情，失败返回错误
func (l *ServiceLogic) GetService(ctx context.Context, id uint) (v1.ServiceResp, error) {
	var service model.Service
	if err := l.svcCtx.DB.First(&service, id).Error; err != nil {
		return v1.ServiceResp{}, err
	}
	return l.modelToResp(&service), nil
}

// DeleteService 删除服务。
//
// 参数:
//   - ctx: 上下文
//   - id: 服务 ID
//
// 返回: 成功返回 nil，失败返回错误
func (l *ServiceLogic) DeleteService(ctx context.Context, id uint) error {
	return l.svcCtx.DB.Delete(&model.Service{}, id).Error
}

// DeployService 部署服务到集群。
//
// 参数:
//   - ctx: 上下文
//   - req: 部署请求，包含服务 ID 和集群 ID
//
// 返回: 成功返回 nil，失败返回错误
func (l *ServiceLogic) DeployService(ctx context.Context, req v1.DeployServiceReq) error {
	var service model.Service
	if err := l.svcCtx.DB.First(&service, req.ServiceID).Error; err != nil {
		return err
	}

	var cluster model.Cluster
	if err := l.svcCtx.DB.First(&cluster, req.ClusterID).Error; err != nil {
		return err
	}

	return DeployToCluster(ctx, &cluster, service.YamlContent)
}

// UpdateService 更新服务配置。
//
// 参数:
//   - ctx: 上下文
//   - id: 服务 ID
//   - req: 更新请求，包含新的服务配置
//
// 返回: 更新后的服务信息，失败返回错误
//
// 流程:
//  1. 查询现有服务
//  2. 重新生成或使用提供的 K8s YAML
//  3. 更新数据库记录
func (l *ServiceLogic) UpdateService(ctx context.Context, id uint, req v1.CreateServiceReq) (v1.ServiceResp, error) {
	var service model.Service
	if err := l.svcCtx.DB.First(&service, id).Error; err != nil {
		return v1.ServiceResp{}, err
	}

	// Regenerate or Use YAML
	var yamlContent string
	var err error

	if req.YamlContent != "" {
		yamlContent = req.YamlContent
	} else {
		yamlContent, err = l.generateK8sYAML(req)
		if err != nil {
			return v1.ServiceResp{}, fmt.Errorf("failed to generate YAML: %v", err)
		}
	}

	var envVarsBytes, resourcesBytes []byte
	if req.EnvVars != nil {
		envVarsBytes, _ = json.Marshal(req.EnvVars)
	}
	if req.Resources != nil {
		resourcesBytes, _ = json.Marshal(req.Resources)
	}

	service.Name = req.Name
	service.Type = req.Type
	service.Image = req.Image
	service.Replicas = req.Replicas
	service.ServicePort = req.ServicePort
	service.ContainerPort = req.ContainerPort
	service.NodePort = req.NodePort
	service.EnvVars = string(envVarsBytes)
	service.Resources = string(resourcesBytes)
	service.YamlContent = yamlContent

	if err := l.svcCtx.DB.Save(&service).Error; err != nil {
		return v1.ServiceResp{}, err
	}

	return l.modelToResp(&service), nil
}

// modelToResp 将模型转换为响应结构。
//
// 参数:
//   - s: 服务模型
//
// 返回: 服务响应结构
func (l *ServiceLogic) modelToResp(s *model.Service) v1.ServiceResp {
	var envVars []v1.EnvVar
	if len(s.EnvVars) > 0 {
		_ = json.Unmarshal([]byte(s.EnvVars), &envVars)
	}
	var resources v1.ResourceReq
	if len(s.Resources) > 0 {
		_ = json.Unmarshal([]byte(s.Resources), &resources)
	}

	return v1.ServiceResp{
		ID:            s.ID,
		ProjectID:     s.ProjectID,
		Name:          s.Name,
		Type:          s.Type,
		Image:         s.Image,
		Replicas:      s.Replicas,
		ServicePort:   s.ServicePort,
		ContainerPort: s.ContainerPort,
		NodePort:      s.NodePort,
		EnvVars:       envVars,
		Resources:     &resources,
		YamlContent:   s.YamlContent,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

// generateK8sYAML 生成 K8s YAML 配置。
//
// 参数:
//   - req: 服务创建请求，包含服务配置
//
// 返回: 生成的 YAML 字符串，失败返回错误
//
// 生成内容:
//   - Deployment 或 StatefulSet (根据 type 字段)
//   - Service (ClusterIP 或 NodePort)
func (l *ServiceLogic) generateK8sYAML(req v1.CreateServiceReq) (string, error) {
	labels := map[string]string{"app": req.Name}
	var replicas int32 = req.Replicas

	// Build EnvVars
	var envs []corev1.EnvVar
	for _, e := range req.EnvVars {
		envs = append(envs, corev1.EnvVar{Name: e.Key, Value: e.Value})
	}

	// Build Resources
	resReq := corev1.ResourceRequirements{
		Limits:   make(corev1.ResourceList),
		Requests: make(corev1.ResourceList),
	}
	if req.Resources != nil {
		if req.Resources.Limits != nil {
			if v, ok := req.Resources.Limits["cpu"]; ok {
				if q, err := resource.ParseQuantity(v); err == nil {
					resReq.Limits[corev1.ResourceCPU] = q
				}
			}
			if v, ok := req.Resources.Limits["memory"]; ok {
				if q, err := resource.ParseQuantity(v); err == nil {
					resReq.Limits[corev1.ResourceMemory] = q
				}
			}
		}
		if req.Resources.Requests != nil {
			if v, ok := req.Resources.Requests["cpu"]; ok {
				if q, err := resource.ParseQuantity(v); err == nil {
					resReq.Requests[corev1.ResourceCPU] = q
				}
			}
			if v, ok := req.Resources.Requests["memory"]; ok {
				if q, err := resource.ParseQuantity(v); err == nil {
					resReq.Requests[corev1.ResourceMemory] = q
				}
			}
		}
	}

	// Container
	container := corev1.Container{
		Name:  req.Name,
		Image: req.Image,
		Ports: []corev1.ContainerPort{
			{ContainerPort: req.ContainerPort},
		},
		Env:       envs,
		Resources: resReq,
	}

	var workload runtime.Object

	if req.Type == "stateful" {
		workload = &appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "StatefulSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: req.Name,
				// Namespace will be set during apply or default to default
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				ServiceName: req.Name,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{container},
					},
				},
			},
		}
	} else {
		workload = &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: req.Name,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{container},
					},
				},
			},
		}
	}

	// Service
	svcObj := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: req.Name,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port:       req.ServicePort,
					TargetPort: intstr.FromInt(int(req.ContainerPort)),
				},
			},
		},
	}
	if req.NodePort > 0 {
		svcObj.Spec.Type = corev1.ServiceTypeNodePort
		svcObj.Spec.Ports[0].NodePort = req.NodePort
	} else {
		svcObj.Spec.Type = corev1.ServiceTypeClusterIP
	}

	// Serializer
	serializer := k8sjson.NewSerializerWithOptions(
		k8sjson.DefaultMetaFactory,
		scheme.Scheme,
		scheme.Scheme,
		k8sjson.SerializerOptions{Yaml: true},
	)

	var buf bytes.Buffer
	if err := serializer.Encode(workload, &buf); err != nil {
		return "", err
	}
	buf.WriteString("\n---\n")
	if err := serializer.Encode(svcObj, &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}
