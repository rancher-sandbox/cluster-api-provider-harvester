package util

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func Healthcheck(config *clientcmdapi.Config) (bool, error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{}

	currentCluster := config.Contexts[config.CurrentContext].Cluster
	currentUser := config.Contexts[config.CurrentContext].AuthInfo
	serverCAstring := config.Clusters[currentCluster].CertificateAuthorityData

	systemTrustedCertificates, err := x509.SystemCertPool()
	if err != nil {
		return false, errors.Wrapf(err, "unable to load system certificate pool")
	}
	//fmt.Println("serverCA :" + string(serverCAstring))
	ok := systemTrustedCertificates.AppendCertsFromPEM(serverCAstring)
	if !ok {
		return false, fmt.Errorf("unable to append CA to Cert pool")
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		RootCAs: systemTrustedCertificates,
	}

	healthcheckUrl := config.Clusters[currentCluster].Server + "/healthz"

	req, err := http.NewRequest("GET", healthcheckUrl, nil)
	if err != nil {
		return false, errors.Wrapf(err, "http request couldn't be create for url: "+healthcheckUrl)
	}

	// TODO: implement scenario where Harvester Cluster Config does not use token but certs
	req.Header.Add("Authorization", "Bearer "+config.AuthInfos[currentUser].Token)

	httpClient := http.Client{}
	resp, err := httpClient.Do(req)

	if err != nil || resp.StatusCode != 200 {
		return false, errors.Wrapf(err, "error during querying Harvester Server")
	}

	var body []byte

	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return false, errors.Wrapf(err, "unable to read response body")
	}

	res := string(body)
	fmt.Println(res)
	if res == "ok" {
		return true, nil
	}

	return false, fmt.Errorf("healthcheck did not respond with 'ok' string")
}
