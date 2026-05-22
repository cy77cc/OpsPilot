package opsagent

import (
	"errors"

	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"gorm.io/gorm"
)

// DBStore implements Store using GORM.
type DBStore struct {
	db *gorm.DB
}

// NewDBStore creates a DBStore.
func NewDBStore(db *gorm.DB) *DBStore {
	return &DBStore{db: db}
}

func (s *DBStore) LoadCA() (certPEM, keyPEM []byte, err error) {
	var ca hostpluginmodel.OpsAgentCA
	if err := s.db.Order("id ASC").First(&ca).Error; err != nil {
		return nil, nil, err
	}
	return []byte(ca.CACertPEM), []byte(ca.CAKeyPEM), nil
}

func (s *DBStore) SaveCA(certPEM, keyPEM []byte) error {
	ca := hostpluginmodel.OpsAgentCA{
		CACertPEM: string(certPEM),
		CAKeyPEM:  string(keyPEM),
	}
	return s.db.Create(&ca).Error
}

// CertStore handles per-host certificate persistence.
type CertStore struct {
	db *gorm.DB
}

// NewCertStore creates a CertStore.
func NewCertStore(db *gorm.DB) *CertStore {
	return &CertStore{db: db}
}

// SaveCert persists a client certificate.
func (cs *CertStore) SaveCert(cert *hostpluginmodel.OpsAgentHostCert) error {
	return cs.db.Create(cert).Error
}

// GetCertByInstance returns the active cert for an instance.
func (cs *CertStore) GetCertByInstance(instanceID uint64) (*hostpluginmodel.OpsAgentHostCert, error) {
	var cert hostpluginmodel.OpsAgentHostCert
	err := cs.db.Where("instance_id = ? AND revoked = ?", instanceID, false).First(&cert).Error
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

// RevokeCert marks a certificate as revoked.
func (cs *CertStore) RevokeCert(instanceID uint64) error {
	result := cs.db.Model(&hostpluginmodel.OpsAgentHostCert{}).
		Where("instance_id = ? AND revoked = ?", instanceID, false).
		Update("revoked", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("no active certificate found for instance")
	}
	return nil
}
