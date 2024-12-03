package util

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/spf13/viper"
)

func GetClientCertificate() (tls.Certificate, error) {
	cert := viper.GetString("tlsClientCert")
	certExists := cert != ""
	key := viper.GetString("tlsClientPrivateKey")
	keyExists := key != ""
	if !certExists && !keyExists {
		return tls.Certificate{}, nil
	}
	if certExists && !keyExists {
		return tls.Certificate{}, fmt.Errorf("Client TLS private key is empty, but client TLS cert was set.")
	}
	if !certExists && keyExists {
		return tls.Certificate{}, fmt.Errorf("Client TLS cert is empty, but client TLS private key was set.")
	}
	return tls.X509KeyPair([]byte(cert), []byte(key))
}

func GetHttpClient() (*http.Client, error) {
	tlsSkipVerify := viper.GetBool("tlsSkipVerify")
	cert, err := GetClientCertificate()
	if err != nil {
		return nil, err
	}
	httpClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: tlsSkipVerify,
			},
		},
	}

	return &httpClient, nil
}
