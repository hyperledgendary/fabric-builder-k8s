// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hyperledgendary/fabric-builder-k8s/internal/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Run struct {
	BuildOutputDirectory string
	RunMetadataDirectory string
	PeerID               string
	KubeconfigPath       string
	KubeNamespace        string
	DevModeTag           string
}

const (
	// Mutual TLS auth client key and cert paths in the chaincode container
	TLSClientKeyPath      string = "/etc/hyperledger/fabric/client.key"
	TLSClientCertPath     string = "/etc/hyperledger/fabric/client.crt"
	TLSClientKeyFile      string = "/etc/hyperledger/fabric/client_pem.key"
	TLSClientCertFile     string = "/etc/hyperledger/fabric/client_pem.crt"
	TLSClientRootCertFile string = "/etc/hyperledger/fabric/peer.crt"
)

func (r *Run) Run() error {
	imageJsonPath := filepath.Join(r.BuildOutputDirectory, "/image.json")
	chaincodeJsonPath := filepath.Join(r.RunMetadataDirectory, "/chaincode.json")

	imageData, err := util.ReadImageJson(imageJsonPath)
	if err != nil {
		return fmt.Errorf("unable to read image.json: %w", err)
	}

	fmt.Println("Reading chaincode.json...")
	_, err = os.Stat(chaincodeJsonPath)
	if err != nil {
		return fmt.Errorf("unable to access chaincode.json: %w", err)
	}

	chaincodeJsonContents, err := ioutil.ReadFile(chaincodeJsonPath)
	if err != nil {
		return fmt.Errorf("unable to read chaincode.json: %w", err)
	}

	var chaincodeData util.ChaincodeJson
	if err := json.Unmarshal(chaincodeJsonContents, &chaincodeData); err != nil {
		return fmt.Errorf("unable to process chaincode.json: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Chaincode ID: %s\n", chaincodeData.ChaincodeID)

	clientset, err := util.GetKubeClientset(r.KubeconfigPath)
	if err != nil {
		return fmt.Errorf("unable to connect kubernetes client: %w", err)
	}

	secretsClient := clientset.CoreV1().Secrets(r.KubeNamespace)

	secret := util.GetChaincodeSecretObject(r.KubeNamespace, r.PeerID, chaincodeData)

	// TODO apply?
	s, err := secretsClient.Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create kubernetes secret: %w", err)
	}
	fmt.Printf("Created secret %s\n", s.Name)

	podsClient := clientset.CoreV1().Pods(r.KubeNamespace)

	pod := util.GetChaincodePodObject(r.DevModeTag, *imageData, r.KubeNamespace, r.PeerID, chaincodeData)

	// TODO apply?
	p, err := podsClient.Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create kubernetes pod: %w", err)
	}
	fmt.Printf("Created pod %s\n", p.Name)

	// TODO watch deployment events instead of returning?

	return nil
}
