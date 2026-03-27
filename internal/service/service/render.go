// Package service 提供服务配置渲染功能。
//
// 本文件实现标准服务配置到 K8s/Docker Compose YAML 的渲染逻辑。
package service

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"

	yamlv3 "gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
)

// renderFromStandard 从标准配置渲染 YAML。
//
// 根据目标平台生成对应的 YAML 配置。
//
// 参数:
//   - name: 服务名称
//   - serviceType: 服务类型 (stateless/stateful)
//   - target: 目标平台 (k8s/compose)
//   - cfg: 标准服务配置
//
// 返回: 渲染预览响应和错误信息
func renderFromStandard(name, serviceType, target string, cfg *StandardServiceConfig) (RenderPreviewResp, error) {
	if cfg == nil {
		return RenderPreviewResp{}, fmt.Errorf("standard_config is required")
	}
	if name == "" {
		name = "sample-service"
	}
	if serviceType == "" {
		serviceType = "stateless"
	}
	if cfg.Replicas <= 0 {
		cfg.Replicas = 1
	}
	if len(cfg.Ports) == 0 {
		cfg.Ports = []PortConfig{{Name: "http", Protocol: "TCP", ContainerPort: 8080, ServicePort: 80}}
	}

	switch target {
	case "compose":
		yml, diags, err := buildComposeYAML(name, cfg)
		if err != nil {
			return RenderPreviewResp{}, err
		}
		return RenderPreviewResp{RenderedYAML: yml, Diagnostics: diags, NormalizedConfig: cfg}, nil
	case "k8s", "":
		yml, diags, err := buildK8sYAML(name, serviceType, cfg)
		if err != nil {
			return RenderPreviewResp{}, err
		}
		return RenderPreviewResp{RenderedYAML: yml, Diagnostics: diags, NormalizedConfig: cfg}, nil
	default:
		return RenderPreviewResp{}, fmt.Errorf("unsupported target: %s", target)
	}
}

// sourceHash 计算内容 SHA256 哈希。
//
// 参数:
//   - content: 输入内容
//
// 返回: 十六进制哈希字符串
func sourceHash(content string) string {
	s := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", s[:])
}

// buildComposeYAML 构建 Docker Compose YAML。
//
// 根据标准配置生成 docker-compose.yaml 格式的配置。
//
// 参数:
//   - name: 服务名称
//   - cfg: 标准服务配置
//
// 返回: YAML 内容、诊断信息和错误
func buildComposeYAML(name string, cfg *StandardServiceConfig) (string, []RenderDiagnostic, error) {
	diags := make([]RenderDiagnostic, 0)
	type composeSvc struct {
		Image       string            `yaml:"image"`
		Ports       []string          `yaml:"ports,omitempty"`
		Environment map[string]string `yaml:"environment,omitempty"`
		Volumes     []string          `yaml:"volumes,omitempty"`
		Healthcheck map[string]any    `yaml:"healthcheck,omitempty"`
	}
	model := map[string]any{
		"name": name,
		"services": map[string]any{
			name: composeSvc{Image: cfg.Image},
		},
	}
	svc := model["services"].(map[string]any)[name].(composeSvc)
	for _, p := range cfg.Ports {
		svc.Ports = append(svc.Ports, fmt.Sprintf("%d:%d", p.ServicePort, p.ContainerPort))
	}
	if len(cfg.Envs) > 0 {
		svc.Environment = map[string]string{}
		for _, e := range cfg.Envs {
			svc.Environment[e.Key] = e.Value
		}
	}
	for _, v := range cfg.Volumes {
		if strings.TrimSpace(v.HostPath) == "" {
			diags = append(diags, RenderDiagnostic{Level: "warning", Code: "compose_volume_hostpath_missing", Message: fmt.Sprintf("volume %s host_path is empty, skipped", v.Name)})
			continue
		}
		svc.Volumes = append(svc.Volumes, fmt.Sprintf("%s:%s", v.HostPath, v.MountPath))
	}
	if cfg.HealthCheck != nil && cfg.HealthCheck.Type == "http" {
		path := cfg.HealthCheck.Path
		if path == "" {
			path = "/health"
		}
		port := cfg.HealthCheck.Port
		if port <= 0 {
			port = cfg.Ports[0].ContainerPort
		}
		svc.Healthcheck = map[string]any{
			"test":     []string{"CMD", "curl", "-f", fmt.Sprintf("http://localhost:%d%s", port, path)},
			"interval": "30s",
			"timeout":  "5s",
			"retries":  3,
		}
	}
	model["services"].(map[string]any)[name] = svc
	b, err := yamlv3.Marshal(model)
	return string(b), diags, err
}

// buildK8sYAML 构建 Kubernetes YAML。
//
// 根据标准配置生成 K8s Deployment/StatefulSet 和 Service 的 YAML 配置。
//
// 参数:
//   - name: 服务名称
//   - serviceType: 服务类型 (stateless/stateful)
//   - cfg: 标准服务配置
//
// 返回: YAML 内容、诊断信息和错误
func buildK8sYAML(name, serviceType string, cfg *StandardServiceConfig) (string, []RenderDiagnostic, error) {
	diags := make([]RenderDiagnostic, 0)
	labels := map[string]string{"app": name}
	replicas := cfg.Replicas

	resources := corev1.ResourceRequirements{Limits: corev1.ResourceList{}, Requests: corev1.ResourceList{}}
	if v, ok := cfg.Resources["cpu"]; ok && strings.TrimSpace(v) != "" {
		if q, err := resource.ParseQuantity(v); err == nil {
			resources.Limits[corev1.ResourceCPU] = q
		} else {
			diags = append(diags, RenderDiagnostic{Level: "warning", Code: "invalid_cpu", Message: err.Error()})
		}
	}
	if v, ok := cfg.Resources["memory"]; ok && strings.TrimSpace(v) != "" {
		if q, err := resource.ParseQuantity(v); err == nil {
			resources.Limits[corev1.ResourceMemory] = q
		} else {
			diags = append(diags, RenderDiagnostic{Level: "warning", Code: "invalid_memory", Message: err.Error()})
		}
	}

	envs := make([]corev1.EnvVar, 0, len(cfg.Envs))
	for _, e := range cfg.Envs {
		envs = append(envs, corev1.EnvVar{Name: e.Key, Value: e.Value})
	}
	ports := make([]corev1.ContainerPort, 0, len(cfg.Ports))
	for _, p := range cfg.Ports {
		ports = append(ports, corev1.ContainerPort{ContainerPort: p.ContainerPort, Protocol: corev1.Protocol(strings.ToUpper(defaultIfEmpty(p.Protocol, "TCP")))})
	}

	volumeMounts := make([]corev1.VolumeMount, 0, len(cfg.Volumes))
	volumes := make([]corev1.Volume, 0, len(cfg.Volumes))
	for _, v := range cfg.Volumes {
		if strings.TrimSpace(v.Name) == "" || strings.TrimSpace(v.MountPath) == "" {
			continue
		}
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: v.Name, MountPath: v.MountPath})
		if strings.TrimSpace(v.HostPath) != "" {
			hp := v.HostPath
			volumes = append(volumes, corev1.Volume{Name: v.Name, VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: hp}}})
		} else {
			volumes = append(volumes, corev1.Volume{Name: v.Name, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
		}
	}

	container := corev1.Container{
		Name:         name,
		Image:        cfg.Image,
		Ports:        ports,
		Env:          envs,
		Resources:    resources,
		VolumeMounts: volumeMounts,
	}
	if cfg.HealthCheck != nil {
		h := cfg.HealthCheck
		if h.Type == "http" {
			path := defaultIfEmpty(h.Path, "/health")
			port := h.Port
			if port <= 0 {
				port = cfg.Ports[0].ContainerPort
			}
			container.ReadinessProbe = &corev1.Probe{
				ProbeHandler:        corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: path, Port: intstr.FromInt(int(port))}},
				InitialDelaySeconds: maxInt32(h.InitialDelaySec, 3),
				PeriodSeconds:       maxInt32(h.PeriodSec, 10),
			}
		}
	}

	var workload runtime.Object
	if serviceType == "stateful" {
		workload = &appsv1.StatefulSet{
			TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "StatefulSet"},
			ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels},
			Spec:       appsv1.StatefulSetSpec{ServiceName: name, Replicas: &replicas, Selector: &metav1.LabelSelector{MatchLabels: labels}, Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}, Spec: corev1.PodSpec{Containers: []corev1.Container{container}, Volumes: volumes}}},
		}
	} else {
		workload = &appsv1.Deployment{
			TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
			ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas, Selector: &metav1.LabelSelector{MatchLabels: labels}, Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}, Spec: corev1.PodSpec{Containers: []corev1.Container{container}, Volumes: volumes}}},
		}
	}

	svcPorts := make([]corev1.ServicePort, 0, len(cfg.Ports))
	for _, p := range cfg.Ports {
		svcPorts = append(svcPorts, corev1.ServicePort{Name: defaultIfEmpty(p.Name, fmt.Sprintf("p%d", p.ServicePort)), Port: p.ServicePort, TargetPort: intstr.FromInt(int(p.ContainerPort)), Protocol: corev1.Protocol(strings.ToUpper(defaultIfEmpty(p.Protocol, "TCP")))})
	}
	svcObj := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels},
		Spec:       corev1.ServiceSpec{Selector: labels, Ports: svcPorts, Type: corev1.ServiceTypeClusterIP},
	}

	serializer := k8sjson.NewSerializerWithOptions(k8sjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme, k8sjson.SerializerOptions{Yaml: true})
	var buf bytes.Buffer
	if err := serializer.Encode(workload, &buf); err != nil {
		return "", nil, err
	}
	buf.WriteString("\n---\n")
	if err := serializer.Encode(svcObj, &buf); err != nil {
		return "", nil, err
	}
	return buf.String(), diags, nil
}

// defaultIfEmpty 返回非空值或默认值。
//
// 参数:
//   - v: 输入值
//   - d: 默认值
//
// 返回: 若输入为空则返回默认值，否则返回输入值
func defaultIfEmpty(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

// maxInt32 返回非零值或默认值。
//
// 参数:
//   - v: 输入值
//   - d: 默认值
//
// 返回: 若输入小于等于零则返回默认值，否则返回输入值
func maxInt32(v, d int32) int32 {
	if v <= 0 {
		return d
	}
	return v
}
