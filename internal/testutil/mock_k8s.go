package testutil

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
)

// MockK8sClient wraps a fake Kubernetes client for testing.
type MockK8sClient struct {
	Clientset *fake.Clientset
	namespace string
}

// NewMockK8sClient creates a new MockK8sClient with a fake clientset.
func NewMockK8sClient() *MockK8sClient {
	return &MockK8sClient{
		Clientset: fake.NewSimpleClientset(),
		namespace: "default",
	}
}

// SetNamespace sets the default namespace for operations.
func (m *MockK8sClient) SetNamespace(ns string) {
	m.namespace = ns
}

// CreateNamespace creates a namespace in the cluster.
func (m *MockK8sClient) CreateNamespace(name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err := m.Clientset.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	return err
}

// DeleteNamespace deletes a namespace from the cluster.
func (m *MockK8sClient) DeleteNamespace(name string) error {
	return m.Clientset.CoreV1().Namespaces().Delete(context.Background(), name, metav1.DeleteOptions{})
}

// GetNamespace retrieves a namespace.
func (m *MockK8sClient) GetNamespace(name string) (*corev1.Namespace, error) {
	return m.Clientset.CoreV1().Namespaces().Get(context.Background(), name, metav1.GetOptions{})
}

// CreatePod creates a pod in the default namespace.
func (m *MockK8sClient) CreatePod(name string, image string, overrides ...func(*corev1.Pod)) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: m.namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  name,
					Image: image,
				},
			},
		},
	}
	for _, fn := range overrides {
		fn(pod)
	}
	return m.Clientset.CoreV1().Pods(m.namespace).Create(context.Background(), pod, metav1.CreateOptions{})
}

// DeletePod deletes a pod from the default namespace.
func (m *MockK8sClient) DeletePod(name string) error {
	return m.Clientset.CoreV1().Pods(m.namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
}

// GetPod retrieves a pod by name.
func (m *MockK8sClient) GetPod(name string) (*corev1.Pod, error) {
	return m.Clientset.CoreV1().Pods(m.namespace).Get(context.Background(), name, metav1.GetOptions{})
}

// ListPods lists all pods in the default namespace.
func (m *MockK8sClient) ListPods() (*corev1.PodList, error) {
	return m.Clientset.CoreV1().Pods(m.namespace).List(context.Background(), metav1.ListOptions{})
}

// ListPodsByNamespace lists all pods in a specific namespace.
func (m *MockK8sClient) ListPodsByNamespace(namespace string) (*corev1.PodList, error) {
	return m.Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
}

// CreateConfigMap creates a configmap in the default namespace.
func (m *MockK8sClient) CreateConfigMap(name string, data map[string]string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: m.namespace,
		},
		Data: data,
	}
	return m.Clientset.CoreV1().ConfigMaps(m.namespace).Create(context.Background(), cm, metav1.CreateOptions{})
}

// GetConfigMap retrieves a configmap by name.
func (m *MockK8sClient) GetConfigMap(name string) (*corev1.ConfigMap, error) {
	return m.Clientset.CoreV1().ConfigMaps(m.namespace).Get(context.Background(), name, metav1.GetOptions{})
}

// CreateSecret creates a secret in the default namespace.
func (m *MockK8sClient) CreateSecret(name string, data map[string][]byte, secretType corev1.SecretType) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: m.namespace,
		},
		Data: data,
		Type: secretType,
	}
	return m.Clientset.CoreV1().Secrets(m.namespace).Create(context.Background(), secret, metav1.CreateOptions{})
}

// GetSecret retrieves a secret by name.
func (m *MockK8sClient) GetSecret(name string) (*corev1.Secret, error) {
	return m.Clientset.CoreV1().Secrets(m.namespace).Get(context.Background(), name, metav1.GetOptions{})
}

// CreateService creates a service in the default namespace.
func (m *MockK8sClient) CreateService(name string, port int32, targetPort int32, svcType corev1.ServiceType) (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: m.namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: svcType,
			Ports: []corev1.ServicePort{
				{
					Port:       port,
					TargetPort: fromInt(targetPort),
				},
			},
		},
	}
	return m.Clientset.CoreV1().Services(m.namespace).Create(context.Background(), svc, metav1.CreateOptions{})
}

// GetService retrieves a service by name.
func (m *MockK8sClient) GetService(name string) (*corev1.Service, error) {
	return m.Clientset.CoreV1().Services(m.namespace).Get(context.Background(), name, metav1.GetOptions{})
}

// CreateDeployment creates a deployment using the apps/v1 API.
// Note: This requires importing the apps/v1 package if needed.
// For simplicity, we use the dynamic client approach or raw YAML.

// ApplyManifest applies a YAML manifest to the cluster.
// This is a simplified implementation that supports basic resources.
func (m *MockK8sClient) ApplyManifest(yamlContent string) error {
	// For a full implementation, you would parse the YAML and create the appropriate resource.
	// This is a placeholder for the interface.
	return nil
}

// Reset clears all resources from the fake client.
func (m *MockK8sClient) Reset() {
	m.Clientset = fake.NewSimpleClientset()
}

// AssertPodExists asserts that a pod exists.
func (m *MockK8sClient) AssertPodExists(t *testing.T, name string) {
	t.Helper()
	pod, err := m.GetPod(name)
	if err != nil {
		t.Fatalf("expected pod %q to exist, but got error: %v", name, err)
	}
	if pod == nil {
		t.Fatalf("expected pod %q to exist, but it was nil", name)
	}
}

// AssertPodNotExists asserts that a pod does not exist.
func (m *MockK8sClient) AssertPodNotExists(t *testing.T, name string) {
	t.Helper()
	_, err := m.GetPod(name)
	if err == nil {
		t.Fatalf("expected pod %q to not exist, but it was found", name)
	}
}

// AssertNamespaceExists asserts that a namespace exists.
func (m *MockK8sClient) AssertNamespaceExists(t *testing.T, name string) {
	t.Helper()
	ns, err := m.GetNamespace(name)
	if err != nil {
		t.Fatalf("expected namespace %q to exist, but got error: %v", name, err)
	}
	if ns == nil {
		t.Fatalf("expected namespace %q to exist, but it was nil", name)
	}
}

// AssertPodCount asserts the number of pods in the namespace.
func (m *MockK8sClient) AssertPodCount(t *testing.T, expected int) {
	t.Helper()
	pods, err := m.ListPods()
	if err != nil {
		t.Fatalf("failed to list pods: %v", err)
	}
	if len(pods.Items) != expected {
		t.Fatalf("expected %d pods, got %d", expected, len(pods.Items))
	}
}

// Helper function for intstr
func fromInt(val int32) intstr.IntOrString {
	return intstr.IntOrString{Type: intstr.Int, IntVal: val}
}

// ============================================================================
// Extended Methods (T2.1.2)
// ============================================================================

// ClusterInfo represents cluster information for testing.
type ClusterInfo struct {
	Name        string
	Version     string
	NodeCount   int
	PodCount    int
	Status      string
}

// ListNodeResult represents a node from ListNodes.
type ListNodeResult struct {
	Name        string
	IP          string
	Role        string
	Status      string
	KubeletVer  string
	Labels      map[string]string
}

// ListNodes returns a list of nodes in the cluster.
func (m *MockK8sClient) ListNodes(ctx context.Context) ([]ListNodeResult, error) {
	// Return mock node data
	return []ListNodeResult{
		{
			Name:       "control-plane-1",
			IP:         "10.0.0.1",
			Role:       "control-plane",
			Status:     "ready",
			KubeletVer: "v1.28.0",
			Labels:     map[string]string{"node-role.kubernetes.io/control-plane": ""},
		},
		{
			Name:       "worker-1",
			IP:         "10.0.0.2",
			Role:       "worker",
			Status:     "ready",
			KubeletVer: "v1.28.0",
			Labels:     map[string]string{},
		},
	}, nil
}

// GetClusterInfo returns cluster information.
func (m *MockK8sClient) GetClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	return &ClusterInfo{
		Name:      "test-cluster",
		Version:   "v1.28.0",
		NodeCount: 2,
		PodCount:  5,
		Status:    "healthy",
	}, nil
}

// HealthCheck performs a health check on the cluster.
func (m *MockK8sClient) HealthCheck(ctx context.Context) error {
	// Always return healthy in mock
	return nil
}

// CreateDeployment creates a deployment in the namespace.
func (m *MockK8sClient) CreateDeployment(ctx context.Context, name, image string, replicas int32, overrides ...func(*appsv1.Deployment)) (*appsv1.Deployment, error) {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: m.namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: name, Image: image},
					},
				},
			},
		},
	}
	for _, fn := range overrides {
		fn(deploy)
	}
	return m.Clientset.AppsV1().Deployments(m.namespace).Create(ctx, deploy, metav1.CreateOptions{})
}

// GetDeployment retrieves a deployment by name.
func (m *MockK8sClient) GetDeployment(name string) (*appsv1.Deployment, error) {
	return m.Clientset.AppsV1().Deployments(m.namespace).Get(context.Background(), name, metav1.GetOptions{})
}

// ListDeployments lists all deployments in the namespace.
func (m *MockK8sClient) ListDeployments() (*appsv1.DeploymentList, error) {
	return m.Clientset.AppsV1().Deployments(m.namespace).List(context.Background(), metav1.ListOptions{})
}
