package framework

import (
	"crypto/rsa"
	"crypto/x509"
	"net"

	"k8s.io/client-go/util/cert"
)

type CertManager struct {
	caKey  *rsa.PrivateKey
	caCert *x509.Certificate
}

func NewCertManager() (*CertManager, error) {
	key, err := cert.NewPrivateKey()
	if err != nil {
		return nil, err
	}
	cfg := cert.Config{
		CommonName: "ca",
	}
	crt, err := cert.NewSelfSignedCACert(cfg, key)
	if err != nil {
		return nil, err
	}
	return &CertManager{caCert: crt, caKey: key}, nil
}

func (cm *CertManager) CACert() []byte {
	return cert.EncodeCertPEM(cm.caCert)
}

func (cm *CertManager) CAKey() []byte {
	return cert.EncodePrivateKeyPEM(cm.caKey)
}

func (cm *CertManager) NewServerCertPair() ([]byte, []byte, error) {
	sans := cert.AltNames{
		IPs:      []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.168.99.100")},
		DNSNames: []string{TestDomain},
	}
	cfg := cert.Config{
		CommonName: "server",
		AltNames:   sans,
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	key, err := cert.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}
	crt, err := cert.NewSignedCert(cfg, key, cm.caCert, cm.caKey)
	if err != nil {
		return nil, nil, err
	}
	return cert.EncodeCertPEM(crt), cert.EncodePrivateKeyPEM(key), nil
}

func (cm *CertManager) NewClientCertPair() ([]byte, []byte, error) {
	cfg := cert.Config{
		CommonName:   "e2e-test",
		Organization: []string{"AppsCode", "Eng"},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	key, err := cert.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}
	crt, err := cert.NewSignedCert(cfg, key, cm.caCert, cm.caKey)
	if err != nil {
		return nil, nil, err
	}
	return cert.EncodeCertPEM(crt), cert.EncodePrivateKeyPEM(key), nil
}
