# Kubernetes Test Network

The k8s builder can be used with the [k8s test network](https://github.com/hyperledger/fabric-samples/tree/main/test-network-k8s) by following the instructions below.

## Configure builder

Before following the instructions to set up the k8s test network, it needs to be configured to use the k8s builder peer image.
Find the latest [k8s-fabric-peer](https://github.com/hyperledgendary/fabric-builder-k8s/pkgs/container/k8s-fabric-peer) image and export a `TEST_NETWORK_FABRIC_PEER_IMAGE` environment variable, e.g.

```shell
export TEST_NETWORK_FABRIC_PEER_IMAGE=ghcr.io/hyperledgendary/k8s-fabric-peer:ac2f9c5288292f69aab91e5556c65d5374697466
```

The org1 and org2 `core.yaml` files also need to be updated with the k8s builder configuration.

```
  externalBuilders:
    - name: k8s_builder
      path: /opt/hyperledger/k8s_builder
      propagateEnvironment:
        - CORE_PEER_ID
        - KUBERNETES_SERVICE_HOST
        - KUBERNETES_SERVICE_PORT
        - FABRIC_K8S_BUILDER_DEV_MODE_TAG
```

You can use [yq](https://mikefarah.gitbook.io/yq/) to update the `core.yaml` files.
Make sure you are in the `fabric-samples/test-network-k8s` directory before running the following commands.

```shell
yq -i '.chaincode.externalBuilders += { "name": "k8s_builder", "path": "/opt/hyperledger/k8s_builder", "propagateEnvironment": [ "CORE_PEER_ID", "KUBERNETES_SERVICE_HOST", "KUBERNETES_SERVICE_PORT", "FABRIC_K8S_BUILDER_DEV_MODE_TAG" ] }' config/org1/core.yaml
yq -i '.chaincode.externalBuilders += { "name": "k8s_builder", "path": "/opt/hyperledger/k8s_builder", "propagateEnvironment": [ "CORE_PEER_ID", "KUBERNETES_SERVICE_HOST", "KUBERNETES_SERVICE_PORT", "FABRIC_K8S_BUILDER_DEV_MODE_TAG" ] }' config/org2/core.yaml
```

Use `kubectl` after running the `./network up` command.

```shell
kubectl patch configmap/org1-peer1-config \
  configmap/org1-peer2-config \
  configmap/org2-peer1-config \
  configmap/org2-peer2-config \
  --namespace=test-network \
  --type merge \
  -p '{"data":{"FABRIC_K8S_BUILDER_DEV_MODE_TAG":"unstable"}}'
kubectl delete pods --namespace=test-network --all
```

_TODO: this should also work an restarts pods automatically but they take a while to come back up due to errors._

```shell
kubectl set env \
  --namespace=test-network \
  deployment/org1-peer1 \
  deployment/org1-peer2 \
  deployment/org2-peer1 \
  deployment/org2-peer2 \
  FABRIC_K8S_BUILDER_DEV_MODE_TAG=unstable
```

## Kubernetes permissions

After launching the k8s test network using the `./network up` command, you also need to configure a k8s service user role to allow the k8s builder to create chaincode deployments.

_TODO: Create a role (cut this down to what is actually required!)_

```shell
cat << ROLE-EOF | kubectl apply -f -
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: fabric-builder-role
  namespace: test-network
rules:
  - apiGroups:
      - ""
      - apps
    resources:
      - pods
      - configmaps
      - secrets
    verbs:
      - get
      - list
      - watch
      - create
      - update
      - patch
      - delete
ROLE-EOF
```

Create a role binding.

```shell
cat << ROLEBINDING-EOF | kubectl apply -f -
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: fabric-builder-rolebinding
  namespace: test-network 
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: fabric-builder-role 
subjects:
- namespace: test-network 
  kind: ServiceAccount
  name: default 
ROLEBINDING-EOF
```

And finally, check it worked!

```shell
kubectl auth can-i create pods --namespace test-network --as system:serviceaccount:test-network:default
kubectl auth can-i create secrets --namespace test-network --as system:serviceaccount:test-network:default
```

## Running peer commands

In the `fabric-samples/test-network-k8s` directory...

Make sure the `build` directory exists, which should be created by the `./network channel create` command.
Then configure the `peer` command environment, e.g. for org1, peer1

```shell
export FABRIC_CFG_PATH=${PWD}/config/org1
export CORE_PEER_ADDRESS=org1-peer1.${TEST_NETWORK_DOMAIN:-vcap.me}:443
export CORE_PEER_MSPCONFIGPATH=${PWD}/build/enrollments/org1/users/org1admin/msp
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/build/channel-msp/peerOrganizations/org1/msp/tlscacerts/tlsca-signcert.pem
```

## Downloading chaincode package

The [conga-nft-contract](https://github.com/hyperledgendary/conga-nft-contract) sample chaincode project publishes a Docker image which the k8s builder can use _and_ a chaincode package file which can be used with the `peer lifecycle chaincode install` command.
This greatly simplifies the deployment process since everything required has been created by a standard build pipeline upfront outside the Fabric environment.

Download the sample chaincode package using `curl`.

```shell
curl -fsSL https://github.com/hyperledgendary/conga-nft-contract/releases/download/v0.1.1/conga-nft-contract-v0.1.1.tgz -o conga-nft-contract-v0.1.1.tgz
```

## Deploying chaincode

Deploy the chaincode package as usual, starting by installing the k8s chaincode package.

```shell
peer lifecycle chaincode install conga-nft-contract-v0.1.1.tgz
```

Export a `PACKAGE_ID` environment variable for use in the following commands.

```shell
export PACKAGE_ID=$(peer lifecycle chaincode calculatepackageid conga-nft-contract-v0.1.1.tgz) && echo $PACKAGE_ID
```

Note: this should match the chaincode code package identifier shown by the `peer lifecycle chaincode install` command.

Approve the chaincode.

```shell
peer lifecycle \
  chaincode approveformyorg \
  --channelID     mychannel \
  --name          conga-nft-contract \
  --version       1 \
  --package-id    ${PACKAGE_ID} \
  --sequence      1 \
  --orderer       org0-orderer1.${TEST_NETWORK_DOMAIN:-vcap.me}:443 \
  --tls --cafile  ${PWD}/build/channel-msp/ordererOrganizations/org0/orderers/org0-orderer1/tls/signcerts/tls-cert.pem
```

Commit the chaincode.

```shell
peer lifecycle \
  chaincode commit \
  --channelID     mychannel \
  --name          conga-nft-contract \
  --version       1 \
  --sequence      1 \
  --orderer       org0-orderer1.${TEST_NETWORK_DOMAIN:-vcap.me}:443 \
  --tls --cafile  ${PWD}/build/channel-msp/ordererOrganizations/org0/orderers/org0-orderer1/tls/signcerts/tls-cert.pem
```

## Running transactions

Query the chaincode metadata!

```shell
./network chaincode query conga-nft-contract '{"Args":["org.hyperledger.fabric:GetMetadata"]}'
```
