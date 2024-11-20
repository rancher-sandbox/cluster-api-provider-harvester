package util

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	"sigs.k8s.io/yaml"

	lbclient "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned"
)

const cloudProviderRoleName = "harvesterhci.io:cloudprovider"

// GetCloudConfigB64 returns the kubeconfig for the service account.
func GetCloudConfigB64(hvClient *lbclient.Clientset, saName string, namespace string, harvesterServerURL string) (string, error) {
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
func createServiceAccountIfNotExists(hvClient *lbclient.Clientset, saName string, namespace string) error {
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
func createClusterRoleBindingIfNotExists(hvClient *lbclient.Clientset, saName string, namespace string) error {
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
func getKubeConfig(hvClient *lbclient.Clientset, saName string, namespace string, harvesterServerURL string) (string, error) {
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
