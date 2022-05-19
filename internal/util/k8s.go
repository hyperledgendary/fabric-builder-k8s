// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	namespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	// Mutual TLS auth client key and cert paths in the chaincode container
	TLSClientKeyPath      string = "/etc/hyperledger/fabric/client.key"
	TLSClientCertPath     string = "/etc/hyperledger/fabric/client.crt"
	TLSClientKeyFile      string = "/etc/hyperledger/fabric/client_pem.key"
	TLSClientCertFile     string = "/etc/hyperledger/fabric/client_pem.crt"
	TLSClientRootCertFile string = "/etc/hyperledger/fabric/peer.crt"
)

var (
	mangledRegExp = regexp.MustCompile("[^a-zA-Z0-9-_.]")
)

// GetKubeClientset returns a client object for a provided kubeconfig filepath
// if one is provided, or which uses the service account kubernetes gives to
// pods otherwise
func GetKubeClientset(kubeconfigPath string) (*kubernetes.Clientset, error) {
	fmt.Fprintf(os.Stdout, "Creating kube client object for kubeconfigPath %s\n", kubeconfigPath)

	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		if kubeconfigPath != "" {
			return nil, fmt.Errorf("unable to load kubeconfig from %s: %w", kubeconfigPath, err)
		}

		return nil, fmt.Errorf("unable to load in-cluster config: %w", err)
	}

	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create a client: %w", err)
	}

	return client, nil
}

func GetKubeNamespace() (string, error) {
	namespace, err := ioutil.ReadFile(namespacePath)
	if err != nil {
		return "", fmt.Errorf("unable to read namespace from %s: %w", namespacePath, err)
	}

	return string(namespace), nil
}

func GetChaincodePodObject(devModeTag string, imageData ImageJson, namespace, peerID string, chaincodeData ChaincodeJson) *apiv1.Pod {
	var chaincodeImage string
	var imagePullPolicy apiv1.PullPolicy
	if devModeTag != "" {
		// Ignore the image digest in dev mode to allow chaincode which has not been pushed to
		// a registry to start
		// TODO log a suitable health warning!
		chaincodeImage = imageData.Name + ":" + devModeTag
		imagePullPolicy = apiv1.PullNever
	} else {
		chaincodeImage = imageData.Name + "@" + imageData.Digest
		imagePullPolicy = apiv1.PullAlways
	}

	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getPodName(chaincodeData.MspID, peerID, chaincodeData.ChaincodeID),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/created-by": "fabric-builder-k8s",
				// TODO name is invalid! Must be no more than 63 characters, etc.
				// "app.kubernetes.io/name":       chaincodeData.ChaincodeID,
			},
		},
		Spec: apiv1.PodSpec{
			Containers: []apiv1.Container{
				{
					Name:            "main",
					Image:           chaincodeImage,
					ImagePullPolicy: imagePullPolicy,
					VolumeMounts: []apiv1.VolumeMount{
						{
							Name:      "certs",
							MountPath: "/etc/hyperledger/fabric",
							ReadOnly:  true,
						},
					},
					Env: []apiv1.EnvVar{
						{
							Name:  "CORE_CHAINCODE_ID_NAME",
							Value: chaincodeData.ChaincodeID,
						},
						{
							Name:  "CORE_PEER_ADDRESS",
							Value: chaincodeData.PeerAddress,
						},
						{
							Name:  "CORE_PEER_TLS_ENABLED",
							Value: "true", // TODO only if there are certs?
						},
						{
							Name:  "CORE_PEER_TLS_ROOTCERT_FILE",
							Value: TLSClientRootCertFile,
						},
						{
							Name:  "CORE_TLS_CLIENT_KEY_PATH",
							Value: TLSClientKeyPath,
						},
						{
							Name:  "CORE_TLS_CLIENT_CERT_PATH",
							Value: TLSClientCertPath,
						},
						{
							Name:  "CORE_TLS_CLIENT_KEY_FILE",
							Value: TLSClientKeyFile,
						},
						{
							Name:  "CORE_TLS_CLIENT_CERT_FILE",
							Value: TLSClientCertFile,
						},
						{
							Name:  "CORE_PEER_LOCALMSPID",
							Value: chaincodeData.MspID,
						},
					},
				},
			},
			Volumes: []apiv1.Volume{
				{
					Name: "certs",
					VolumeSource: apiv1.VolumeSource{
						Secret: &apiv1.SecretVolumeSource{
							SecretName:  getSecretName(chaincodeData.MspID, peerID, chaincodeData.ChaincodeID),
						},
					},
				},
			},
		},
	}
}

func GetChaincodeSecretObject(namespace, peerID string, chaincodeData ChaincodeJson) *apiv1.Secret {
	return &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSecretName(chaincodeData.MspID, peerID, chaincodeData.ChaincodeID),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/created-by": "fabric-builder-k8s",
				// TODO name is invalid! Must be no more than 63 characters, etc.
				// "app.kubernetes.io/name":       chaincodeData.ChaincodeID,
			},
		},
		Type: apiv1.SecretTypeOpaque,
		StringData: map[string]string{
			"peer.crt":       chaincodeData.RootCert,
			"client_pem.crt": chaincodeData.ClientCert,
			"client_pem.key": chaincodeData.ClientKey,
			"client.crt":     base64.StdEncoding.EncodeToString([]byte(chaincodeData.ClientCert)),
			"client.key":     base64.StdEncoding.EncodeToString([]byte(chaincodeData.ClientKey)),
		},
	}
}

func getPodName(mspID, peerID, chaincodeID string) string {
	return mangleName("cc-" + mspID + "-" + peerID + chaincodeID)
}

func getSecretName(mspID, peerID, chaincodeID string) string {
	return mangleName("cc-" + mspID + "-" + peerID + chaincodeID)
}

func mangleName(name string) string {
	// TODO need sensible unique naming scheme for deployments and secrets!
	return strings.ToLower(mangledRegExp.ReplaceAllString(name, "-")[:63])
}
