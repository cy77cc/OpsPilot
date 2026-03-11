// Package pki 提供 PKI 证书管理功能。
//
// 本文件实现 PKI 管理器，用于管理 Kubernetes 集群的 CA 证书、
// 节点证书和 kubeconfig 文件。
package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// PKI 文件名常量。
const (
	CACertName            = "ca.crt"            // Kubernetes CA 证书
	CAKeyName             = "ca.key"            // Kubernetes CA 私钥
	EtcdCACertName        = "etcd/ca.crt"       // etcd CA 证书
	EtcdCAKeyName         = "etcd/ca.key"       // etcd CA 私钥
	FrontProxyCAName      = "front-proxy-ca.crt" // 前端代理 CA 证书
	FrontProxyCAKeyName   = "front-proxy-ca.key" // 前端代理 CA 私钥
	ServiceAccountKeyName = "sa.key"            // Service Account 私钥
	ServiceAccountPubName = "sa.pub"            // Service Account 公钥
)

// Manager 是 PKI 管理器，管理证书的生成和存储。
type Manager struct {
	PKIPath           string // PKI 目录路径
	APIServerEndpoint string // API Server 端点 (如 "192.168.1.10:6443")
	ClusterDomain     string // 集群域名 (如 "cluster.local")
	ServiceCIDR       string // Service CIDR (如 "10.96.0.0/12")
}

// NewManager 创建 PKI 管理器实例。
func NewManager(pkiPath, apiEndpoint, clusterDomain, serviceCIDR string) *Manager {
	return &Manager{
		PKIPath:           pkiPath,
		APIServerEndpoint: apiEndpoint,
		ClusterDomain:     clusterDomain,
		ServiceCIDR:       serviceCIDR,
	}
}

// EnsureCAs 确保所有 CA 证书存在。
//
// 创建 Kubernetes CA、etcd CA、前端代理 CA 和 Service Account 密钥对。
func (m *Manager) EnsureCAs() error {
	// K8s CA
	if err := m.ensureCA("kubernetes-ca", m.PKIPath, CACertName, CAKeyName); err != nil {
		return err
	}
	// Etcd CA
	if err := m.ensureCA("etcd-ca", filepath.Join(m.PKIPath, "etcd"), "ca.crt", "ca.key"); err != nil {
		return err
	}
	// Front Proxy CA
	if err := m.ensureCA("kubernetes-front-proxy-ca", m.PKIPath, FrontProxyCAName, FrontProxyCAKeyName); err != nil {
		return err
	}
	// Service Account Keys
	if err := m.ensureSAKeys(); err != nil {
		return err
	}
	return nil
}

// ensureCA 确保指定的 CA 证书和私钥存在。
func (m *Manager) ensureCA(cn, dir, certName, keyName string) error {
	certPath := filepath.Join(dir, certName)
	keyPath := filepath.Join(dir, keyName)

	if fileExists(certPath) && fileExists(keyPath) {
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	_, _, certPEM, keyPEM, err := GenerateCA(CertSpec{
		CommonName: cn,
		ValidYears: 10,
	})
	if err != nil {
		return err
	}

	if err := WriteFile(certPath, certPEM); err != nil {
		return err
	}
	if err := WriteFile(keyPath, keyPEM); err != nil {
		return err
	}
	return nil
}

// ensureSAKeys 确保 Service Account 密钥对存在。
func (m *Manager) ensureSAKeys() error {
	keyPath := filepath.Join(m.PKIPath, ServiceAccountKeyName)
	pubPath := filepath.Join(m.PKIPath, ServiceAccountPubName)

	if fileExists(keyPath) && fileExists(pubPath) {
		return nil
	}

	// Generate RSA Key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	pubDER, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	if err := WriteFile(keyPath, keyPEM); err != nil {
		return err
	}
	if err := WriteFile(pubPath, pubPEM); err != nil {
		return err
	}
	return nil
}

// CreateMasterCerts 为 Master 节点生成证书。
//
// 包括：kube-apiserver、apiserver-kubelet-client、front-proxy-client、
// etcd server/peer/healthcheck、apiserver-etcd-client 以及 kubeconfig 文件。
func (m *Manager) CreateMasterCerts(nodeName, nodeIP string) error {
	caCert, caKey, err := m.loadCA(filepath.Join(m.PKIPath, CACertName), filepath.Join(m.PKIPath, CAKeyName))
	if err != nil {
		return err
	}
	etcdCACert, etcdCAKey, err := m.loadCA(filepath.Join(m.PKIPath, "etcd/ca.crt"), filepath.Join(m.PKIPath, "etcd/ca.key"))
	if err != nil {
		return err
	}
	fpCACert, fpCAKey, err := m.loadCA(filepath.Join(m.PKIPath, FrontProxyCAName), filepath.Join(m.PKIPath, FrontProxyCAKeyName))
	if err != nil {
		return err
	}

	// API Server
	// SANs: NodeIP, Localhost, ServiceIP, kubernetes...
	svcIP := m.getServiceIP()
	sans := []string{
		nodeName,
		"kubernetes",
		"kubernetes.default",
		"kubernetes.default.svc",
		fmt.Sprintf("kubernetes.default.svc.%s", m.ClusterDomain),
		"127.0.0.1",
		nodeIP,
		svcIP.String(),
	}

	// Add APIServerEndpoint IP if it's an IP
	host, _, _ := net.SplitHostPort(m.APIServerEndpoint)
	if net.ParseIP(host) != nil {
		sans = append(sans, host)
	}

	// kube-apiserver
	if err := m.issueAndWrite(caCert, caKey, CertSpec{
		CommonName:  "kube-apiserver",
		DNSNames:    sans,
		IPs:         parseIPs(sans),
		ValidYears:  1,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}, "apiserver"); err != nil {
		return err
	}

	// kubelet-client (for apiserver to talk to kubelet)
	if err := m.issueAndWrite(caCert, caKey, CertSpec{
		CommonName:  "kube-apiserver-kubelet-client",
		Orgs:        []string{"system:masters"},
		ValidYears:  1,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, "apiserver-kubelet-client"); err != nil {
		return err
	}

	// front-proxy-client
	if err := m.issueAndWrite(fpCACert, fpCAKey, CertSpec{
		CommonName:  "front-proxy-client",
		ValidYears:  1,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, "front-proxy-client"); err != nil {
		return err
	}

	// Etcd Server
	if err := m.issueAndWrite(etcdCACert, etcdCAKey, CertSpec{
		CommonName:  "etcd-" + nodeName, // Unique per node
		DNSNames:    []string{nodeName, "localhost"},
		IPs:         []net.IP{net.ParseIP(nodeIP), net.ParseIP("127.0.0.1")},
		ValidYears:  1,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}, "etcd/server"); err != nil {
		return err
	}

	// Etcd Peer
	if err := m.issueAndWrite(etcdCACert, etcdCAKey, CertSpec{
		CommonName:  "etcd-" + nodeName,
		DNSNames:    []string{nodeName, "localhost"},
		IPs:         []net.IP{net.ParseIP(nodeIP), net.ParseIP("127.0.0.1")},
		ValidYears:  1,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}, "etcd/peer"); err != nil {
		return err
	}

	// Etcd Healthcheck Client
	if err := m.issueAndWrite(etcdCACert, etcdCAKey, CertSpec{
		CommonName:  "kube-etcd-healthcheck-client",
		Orgs:        []string{"system:masters"},
		ValidYears:  1,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, "etcd/healthcheck-client"); err != nil {
		return err
	}

	// API Server Etcd Client
	if err := m.issueAndWrite(etcdCACert, etcdCAKey, CertSpec{
		CommonName:  "kube-apiserver-etcd-client",
		Orgs:        []string{"system:masters"},
		ValidYears:  1,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, "apiserver-etcd-client"); err != nil {
		return err
	}

	// Generate Kubeconfigs
	if err := m.createKubeConfig(caCert, caKey, "admin", []string{"system:masters"}, "admin.conf"); err != nil {
		return err
	}
	if err := m.createKubeConfig(caCert, caKey, "system:kube-controller-manager", nil, "controller-manager.conf"); err != nil {
		return err
	}
	if err := m.createKubeConfig(caCert, caKey, "system:kube-scheduler", nil, "scheduler.conf"); err != nil {
		return err
	}

	return nil
}

// CreateWorkerCerts 为 Worker 节点生成证书。
//
// 主要生成 kubelet 的 kubeconfig 文件。
func (m *Manager) CreateWorkerCerts(nodeName string) error {
	caCert, caKey, err := m.loadCA(filepath.Join(m.PKIPath, CACertName), filepath.Join(m.PKIPath, CAKeyName))
	if err != nil {
		return err
	}

	// Kubelet Client Cert (for Kubelet to talk to API Server)
	// Usually bootstrapping is used, but manual is fine too.
	cn := fmt.Sprintf("system:node:%s", nodeName)
	if err := m.createKubeConfig(caCert, caKey, cn, []string{"system:nodes"}, fmt.Sprintf("kubelet-%s.conf", nodeName)); err != nil {
		return err
	}
	return nil
}

// =============================================================================
// 辅助方法
// =============================================================================

// loadCA 加载 CA 证书和私钥。
func (m *Manager) loadCA(certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, err
	}
	cert, err := ParseCert(certData)
	if err != nil {
		return nil, nil, err
	}
	key, err := ParseRSAKey(keyData)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

// issueAndWrite 签发证书并写入文件。
func (m *Manager) issueAndWrite(caCert *x509.Certificate, caKey *rsa.PrivateKey, spec CertSpec, name string) error {
	certPEM, keyPEM, err := IssueCert(caCert, caKey, spec)
	if err != nil {
		return err
	}
	if err := WriteFile(filepath.Join(m.PKIPath, name+".crt"), certPEM); err != nil {
		return err
	}
	if err := WriteFile(filepath.Join(m.PKIPath, name+".key"), keyPEM); err != nil {
		return err
	}
	return nil
}

// createKubeConfig 创建 kubeconfig 文件。
func (m *Manager) createKubeConfig(caCert *x509.Certificate, caKey *rsa.PrivateKey, cn string, orgs []string, fileName string) error {
	certPEM, keyPEM, err := IssueCert(caCert, caKey, CertSpec{
		CommonName:  cn,
		Orgs:        orgs,
		ValidYears:  1,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		return err
	}

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw})

	config := fmt.Sprintf(`apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: %s
    server: https://%s
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: %s
  name: %s@kubernetes
current-context: %s@kubernetes
kind: Config
preferences: {}
users:
- name: %s
  user:
    client-certificate-data: %s
    client-key-data: %s
`,
		base64.StdEncoding.EncodeToString(caPEM),
		m.APIServerEndpoint,
		cn, cn, cn, cn,
		base64.StdEncoding.EncodeToString(certPEM),
		base64.StdEncoding.EncodeToString(keyPEM),
	)

	return WriteFile(filepath.Join(m.PKIPath, fileName), []byte(config))
}

// getServiceIP 获取 Service CIDR 中的第一个 IP。
func (m *Manager) getServiceIP() net.IP {
	ip, _, err := net.ParseCIDR(m.ServiceCIDR)
	if err != nil {
		return net.ParseIP("10.96.0.1") // Default fallback
	}
	// Typically the first IP is the service IP.
	// Just increment the last byte for simplicity or use logic to get 1st IP
	// For strictness, we should use a proper IP increment function.
	// But assuming standard /12 or /16, mostly ends with .1
	// Let's just return the IP from ParseCIDR which is the network address, and add 1.
	// A simple hack for now:
	ip[len(ip)-1]++
	return ip
}

// parseIPs 从字符串列表中解析 IP 地址。
func parseIPs(sans []string) []net.IP {
	var ips []net.IP
	for _, s := range sans {
		if ip := net.ParseIP(s); ip != nil {
			ips = append(ips, ip)
		}
	}
	return ips
}

// fileExists 检查文件是否存在。
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
