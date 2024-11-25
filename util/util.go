package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	regen "github.com/zach-klippenstein/goregen"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/rancher-sandbox/cluster-api-provider-harvester/api/v1alpha1"
	hvclientset "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned"
)

const (
	ConfigSecretDataKey = "kubeconfig"
	maximumLabelLength  = 63
)

func Healthcheck(config *clientcmdapi.Config) (bool, error) {
	httpTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return false, fmt.Errorf("unable to assert http.DefaultTransport as *http.Transport")
	}

	httpTransport.TLSClientConfig = &tls.Config{}

	currentCluster := config.Contexts[config.CurrentContext].Cluster
	currentUser := config.Contexts[config.CurrentContext].AuthInfo
	serverCAstring := config.Clusters[currentCluster].CertificateAuthorityData

	systemTrustedCertificates, err := x509.SystemCertPool()
	if err != nil {
		return false, errors.Wrapf(err, "unable to load system certificate pool")
	}
	// fmt.Println("serverCA :" + string(serverCAstring))
	ok = systemTrustedCertificates.AppendCertsFromPEM(serverCAstring)
	if !ok {
		return false, fmt.Errorf("unable to append CA to Cert pool")
	}

	httpTransport.TLSClientConfig = &tls.Config{
		RootCAs: systemTrustedCertificates,
	}

	healthcheckUrl := config.Clusters[currentCluster].Server + "/healthz"

	req, err := http.NewRequest(http.MethodGet, healthcheckUrl, nil)
	if err != nil {
		return false, errors.Wrapf(err, "http request couldn't be create for url: "+healthcheckUrl)
	}

	req.Header.Add("Authorization", "Bearer "+config.AuthInfos[currentUser].Token)

	httpClient := http.Client{}
	resp, err := httpClient.Do(req)

	if err != nil || resp.StatusCode != http.StatusOK {
		return false, errors.Wrapf(err, "error during querying Harvester Server")
	}

	var body []byte

	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return false, errors.Wrapf(err, "unable to read response body")
	}

	res := string(body)
	// fmt.Println(res)
	if res == "ok" {
		return true, nil
	}

	return false, fmt.Errorf("healthcheck did not respond with 'ok' string")
}

func GetSecretForHarvesterConfig(ctx context.Context, cluster *infrav1.HarvesterCluster, cl client.Client) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey(cluster.Spec.IdentitySecret)

	err := cl.Get(ctx, secretKey, secret, &client.GetOptions{})

	return secret, err
}

func GetHarvesterClientFromSecret(secret *corev1.Secret) (*hvclientset.Clientset, error) {
	hvRESTConfig, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[ConfigSecretDataKey])
	if err != nil {
		return &hvclientset.Clientset{}, err
	}

	return hvclientset.NewForConfig(hvRESTConfig)
}

// RandomID returns a random string used as an ID internally in Harvester.
func RandomID() string {
	res, err := regen.Generate("[a-z]{3}[0-9][a-z]")
	if err != nil {
		return ""
	}

	return res
}

// NewTrue returns a pointer to true.
func NewTrue() *bool {
	b := true

	return &b
}

// Filter is a generic filter function.
func Filter[T any](ss []T, test func(T) bool) (ret []T) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}

	return
}

// CheckNamespacedName checks if the given string is in the format of "namespace/name".
func CheckNamespacedName(name string) bool {
	return regexp.MustCompile(`^[a-z0-9-\.]+/[a-z0-9-\.]+$`).MatchString(name)
}

// GetNamespacedName returns the namespace and name from the given string in the format of "namespace/name".
func GetNamespacedName(name string, alternativeTargetNS string) (error, types.NamespacedName) {
	// If the given string is in the format of "namespace/name", return the namespace and name from the string.
	if CheckNamespacedName(name) {
		s := strings.Split(name, "/")

		return nil, types.NamespacedName{
			Namespace: s[0],
			Name:      s[1],
		}
	}

	if !regexp.MustCompile(`^[a-z0-9-\.]+$`).MatchString(name) {
		return fmt.Errorf("malformed reference, should be <NAMESPACE>/<NAME>"), types.NamespacedName{}
	}

	// Else, return the namespace from the ownerObject and the name from the string.
	return nil, types.NamespacedName{
		Namespace: alternativeTargetNS,
		Name:      name,
	}
}

// GenerateRFC1035Name generates a name that is RFC-1035 compliant from an array of strings separated by dashes.
func GenerateRFC1035Name(nameComponents []string) string {
	// Join the components with a dash
	name := strings.Join(nameComponents, "-")

	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace any invalid characters with a dash
	re := regexp.MustCompile(`[^a-z0-9-]`)
	name = re.ReplaceAllString(name, "-")

	// Trim leading and trailing dashes
	name = strings.Trim(name, "-")

	// Ensure the name starts with a letter
	if len(name) > 0 && (name[0] < 'a' || name[0] > 'z') {
		name = "a-" + name
	}

	// Truncate to 63 characters
	if len(name) > maximumLabelLength {
		name = name[:maximumLabelLength]
	}

	return name
}

// ValidateB64Kubeconfig validates a base64 encoded kubeconfig.
func ValidateB64Kubeconfig(kubeconfigB64 string) error {
	kubeconfigBytes, err := base64.StdEncoding.DecodeString(kubeconfigB64)
	if err != nil {
		return err
	}

	clientConfigFromBinary, err := clientcmd.Load(kubeconfigBytes)
	if err != nil {
		return err
	}

	return clientcmd.Validate(*clientConfigFromBinary)
}
