// Package pki 提供 PKI 证书管理功能。
//
// 本文件实现 CA 证书生成、终端证书签发、证书解析等功能，
// 用于 Kubernetes 集群的证书管理。
package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

// CertSpec 是证书规格定义。
type CertSpec struct {
	CommonName string          // 通用名称 (CN)
	Orgs       []string        // 组织 (O)

	DNSNames []string          // DNS 名称列表
	IPs      []net.IP          // IP 地址列表

	IsCA       bool            // 是否为 CA 证书
	ValidYears int             // 有效年限

	KeyUsage    x509.KeyUsage     // 密钥用途
	ExtKeyUsage []x509.ExtKeyUsage // 扩展密钥用途
}

// GenerateCA 生成自签名 CA 证书。
//
// 返回证书对象、私钥、PEM 编码的证书和私钥。
func GenerateCA(spec CertSpec) (*x509.Certificate, *rsa.PrivateKey, []byte, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName:   spec.CommonName,
			Organization: spec.Orgs,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(spec.ValidYears, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	cert, _ := x509.ParseCertificate(der)
	return cert, key, certPEM, keyPEM, nil
}

// IssueCert 使用 CA 证书签发终端证书。
//
// 典型用途：
//   - kube-apiserver: ExtKeyUsage = serverAuth, SAN 包含所有 master IP、LB、kubernetes 等
//   - kubelet: CN = system:node:<nodeName>, O = system:nodes
//   - controller/scheduler: CN = system:kube-controller-manager / system:kube-scheduler
//   - kubectl: CN = admin, O = system:masters
func IssueCert(
	caCert *x509.Certificate,
	caKey *rsa.PrivateKey,
	spec CertSpec,
) ([]byte, []byte, error) {

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName:   spec.CommonName,
			Organization: spec.Orgs,
		},
		DNSNames:    spec.DNSNames,
		IPAddresses: spec.IPs,

		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(spec.ValidYears, 0, 0),

		KeyUsage:    spec.KeyUsage,
		ExtKeyUsage: spec.ExtKeyUsage,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	return certPEM, keyPEM, nil
}

// WriteFile 写入文件，权限为 0600。
func WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0600)
}

// ParseCert 解析 PEM 编码的证书。
func ParseCert(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse certificate PEM")
	}
	return x509.ParseCertificate(block.Bytes)
}

// ParseRSAKey 解析 PEM 编码的 RSA 私钥。
func ParseRSAKey(pemData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse key PEM")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
