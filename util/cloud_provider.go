package util

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	re "regexp"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	machineryyaml "k8s.io/apimachinery/pkg/util/yaml"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	"sigs.k8s.io/yaml"

	lbclient "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned"
)

const (
	readerBufferSize      = 4096
	cloudProviderRoleName = "harvesterhci.io:cloudprovider"
	maxNumberOfSecrets    = 15
)

// GetCloudConfigB64 returns the kubeconfig for the service account.
func GetCloudConfigB64(hvClient lbclient.Interface, saName string, namespace string, harvesterServerURL string) (string, error) {
	err := createServiceAccountIfNotExists(hvClient, saName, namespace)
	if err != nil {
		return "", err
	}

	err = createClusterRoleBindingIfNotExists(hvClient, saName, namespace)
	if err != nil {
		return "", err
	}

	kubeconfig, err := getKubeConfig(hvClient, saName, namespace, harvesterServerURL)

	return kubeconfig, err
}

// createServiceAccountIfNotExists creates a service account if it does not exist.
func createServiceAccountIfNotExists(hvClient lbclient.Interface, saName string, namespace string) error {
	_, err := hvClient.CoreV1().ServiceAccounts(namespace).Get(context.Background(), saName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		serviceAccount := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: saName,
			},
		}

		_, err := hvClient.CoreV1().ServiceAccounts(namespace).Create(context.Background(), serviceAccount, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// createClusterRoleBindingIfNotExists creates a cluster role binding for the Cloud Provider's ServiceAccount if it does not exist.
func createClusterRoleBindingIfNotExists(hvClient lbclient.Interface, saName string, namespace string) error {
	_, err := hvClient.RbacV1().ClusterRoleBindings().Get(context.Background(), saName, metav1.GetOptions{})
	if err == nil {
		return nil
	} else if !apierrors.IsNotFound(err) {
		return err
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: saName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: cloudProviderRoleName,
		},
	}

	_, err = hvClient.RbacV1().ClusterRoleBindings().Create(context.Background(), clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return err
}

// getKubeConfig returns a kubeconfig from the Secret associated with the ServiceAccount.
func getKubeConfig(hvClient lbclient.Interface, saName string, namespace string, harvesterServerURL string) (string, error) {
	sa, err := hvClient.CoreV1().ServiceAccounts(namespace).Get(context.Background(), saName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	serviceAccountUID := string(sa.UID)
	serviceAccountName := sa.Name
	secretName := fmt.Sprintf("%s-token", serviceAccountName)

	// Create a secret for the service account
	_, err = hvClient.CoreV1().Secrets(namespace).Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
			Annotations: map[string]string{
				corev1.ServiceAccountNameKey: serviceAccountName,
				corev1.ServiceAccountUIDKey:  serviceAccountUID,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ServiceAccount",
					Name:       serviceAccountName,
					UID:        sa.UID,
				},
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return "", err
	}

	time.Sleep(time.Second)

	secret, err := hvClient.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// Get Endpoint from Service
	vipSVC, err := hvClient.CoreV1().Services("kube-system").Get(context.Background(), "ingress-expose", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("Unable to compute the Harvester Endpoint: problem in getting the ingress-expose service: %v", err)
	}

	vipIP := vipSVC.Annotations["kube-vip.io/loadbalancerIPs"]

	if ok, err := re.MatchString(`\d+\.\d+\.\d+\.\d+`, vipIP); ok && err == nil {
		harvesterServerURL = fmt.Sprintf("https://%s:6443", vipIP)
	}

	kubeconfig, err := buildKubeconfigFromSecret(secret, namespace, harvesterServerURL)
	if err != nil {
		return "", fmt.Errorf("unable to build a kubeconfig from secret %s", saName)
	}

	return base64.StdEncoding.EncodeToString([]byte(kubeconfig)), nil
}

// buildKubeconfigFromSecret builds a kubeconfig from a secret content.
func buildKubeconfigFromSecret(secret *corev1.Secret, namespace string, harvesterServerURL string) (string, error) {
	token, ok := secret.Data[corev1.ServiceAccountTokenKey]
	if !ok {
		return "", fmt.Errorf("token not found in secret")
	}

	ca, ok := secret.Data[corev1.ServiceAccountRootCAKey]
	if !ok {
		return "", fmt.Errorf("ca.crt not found in secret")
	}

	kubeconfigObject := &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"default": {
				Server:                   harvesterServerURL,
				CertificateAuthorityData: ca,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"default": {
				Token: string(token),
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"context": {
				Cluster:   "default",
				AuthInfo:  "default",
				Namespace: namespace,
			},
		},
		CurrentContext: "context",
	}

	jsonConfig, err := runtime.Encode(clientcmdlatest.Codec, kubeconfigObject)
	if err != nil {
		return "", fmt.Errorf("unable to encode kubeconfig object")
	}

	yamlConfig, err := yaml.JSONToYAML(jsonConfig)
	if err != nil {
		return "", fmt.Errorf("unable to convert JSON to YAML: %v", err)
	}

	return string(yamlConfig), nil
}

// GetDataKeyFromConfigMap returns the data key from a ConfigMap.
func GetDataKeyFromConfigMap(configMap *corev1.ConfigMap, key string) (string, error) {
	data, ok := configMap.Data[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in configmap %s", key, configMap.Name)
	}

	return data, nil
}

// // GetConfigMap returns a ConfigMap from the given namespaced name.
// func GetConfigMap(client client.Client, namespace string, name string) (*corev1.ConfigMap, error) {
// 	return client.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, metav1.GetOptions{})
// }

// GetSerializedObjects returns an array of serialized objects from YAML string.
func GetSerializedObjects(yamlString string) ([]runtime.RawExtension, error) {
	decoder := machineryyaml.NewYAMLOrJSONDecoder(
		strings.NewReader(yamlString),
		readerBufferSize,
	)

	var objects []runtime.RawExtension

	for {
		var obj runtime.RawExtension

		err := decoder.Decode(&obj)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}

			return nil, err
		}

		objects = append(objects, obj)
	}

	return objects, nil
}

// GetSecrets returns a list of ConfigMaps from a list of serialized objects.
func GetSecrets(objects []runtime.RawExtension) ([]*corev1.Secret, []int, error) {
	secrets := []*corev1.Secret{}
	indexes := make([]int, 0, maxNumberOfSecrets)

	var secret *corev1.Secret

	for i, obj := range objects {
		secret = &corev1.Secret{}

		var unstructuredObj unstructured.Unstructured

		err := json.Unmarshal(obj.Raw, &unstructuredObj)
		if err != nil {
			continue
		}

		if unstructuredObj.GetKind() != "Secret" || unstructuredObj.GetAPIVersion() != "v1" {
			continue
		}

		err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, secret)
		if err != nil {
			continue
		}

		indexes = append(indexes, i)
		secrets = append(secrets, secret)
	}

	return secrets, indexes, nil
}

// FindSecretByName returns a Secret from a list of Secret by name and namespace.
func FindSecretByName(secrets []*corev1.Secret, name string, namespace string) (*corev1.Secret, int, error) {
	for i, secret := range secrets {
		if secret.Name == name && secret.Namespace == namespace {
			return secret, i, nil
		}
	}

	return nil, 0, fmt.Errorf("secret %s not found in namespace %s", name, namespace)
}

// SetSecretData sets the data of a Secret.
func SetSecretData(secret *corev1.Secret, key string, value []byte) {
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}

	secret.Data[key] = value
}

// SetObjectByIndex sets an object in a list of serialized objects by index.
func SetObjectByIndex(objects []runtime.RawExtension, index int, obj runtime.RawExtension) {
	objects[index] = obj
}

// ModifyYAMlString modifies a YAML string by replacing the value of a key in a Secret.
func ModifyYAMlString(yamlString string, secretName string, secretNamespace string, key string, value []byte) (string, error) {
	objects, err := GetSerializedObjects(yamlString)
	if err != nil {
		return "", err
	}

	secrets, indexes, err := GetSecrets(objects)
	if err != nil {
		return "", err
	}

	secret, index, err := FindSecretByName(secrets, secretName, secretNamespace)
	if err != nil {
		return "", err
	}

	SetSecretData(secret, key, value)

	secretBytes, err := json.Marshal(secret)
	if err != nil {
		return "", err
	}

	SetObjectByIndex(objects, indexes[index], runtime.RawExtension{
		Object: secret,
		Raw:    secretBytes,
	})

	var yamlStrings []string // nolint: prealloc

	for _, obj := range objects {
		yamlBytes, err := yaml.JSONToYAML(obj.Raw)
		if err != nil {
			return "", err
		}

		yamlStrings = append(yamlStrings, string(yamlBytes))
	}

	return strings.Join(yamlStrings, "\n---\n"), nil
}
