# contributing

Code is based on [Digital Ocean CSI driver](https://github.com/digitalocean/csi-digitalocean)

Install [Golang](https://golang.org/)

Run: `go run main.go --endpoint unix://.csi.sock --workdir .workdir`

## Using grpcurl for local testing

[Install grpcurl](https://github.com/fullstorydev/grpcurl)

List / describe methods

```
grpcurl -plaintext -unix .csi.sock list
grpcurl -plaintext -unix .csi.sock describe csi.v1.Identity
grpcurl -plaintext -unix .csi.sock describe csi.v1.Node
```

Identity methods

```
grpcurl -plaintext -unix .csi.sock csi.v1.Identity/GetPluginCapabilities
grpcurl -plaintext -unix .csi.sock csi.v1.Identity/Probe
grpcurl -plaintext -unix .csi.sock csi.v1.Identity/GetPluginInfo
```

Node methods

```
grpcurl -plaintext -unix .csi.sock csi.v1.Node/NodeGetCapabilities
grpcurl -plaintext -unix .csi.sock csi.v1.Node/NodeGetInfo

grpcurl -plaintext \
    -d '{"volumeId": "my-test-volume", "targetPath": ".targetPath", "volume_context": {"mountScript": "echo mount TARGET_PATH=$TARGET_PATH VOLUME_ID=$VOLUME_ID", "unmountScript": "echo unmount TARGET_PATH=$TARGET_PATH VOLUME_ID=$VOLUME_ID"}}' \
    -unix .csi.sock csi.v1.Node/NodePublishVolume

grpcurl -plaintext \
    -d '{"volumeId": "my-test-volume", "targetPath": ".targetPath"}' \
    -unix .csi.sock csi.v1.Node/NodeUnpublishVolume
```

## Testing on Minikube

Start minikube

Connect to minikube's docker env

```
eval $(minikube docker-env)
```

Build:

```
VERSION=dev
COMMIT=$(shell git rev-parse HEAD)
if [ "$(git status --porcelain 2>/dev/null)" == "" ]; then
    GIT_TREE_STATE=clean
else
    GIT_TREE_STATE=dirty
fi
LDFLAGS="-X kamatera/shell-script-csi-driver/driver.version=${VERSION} -X kamatera/shell-script-csi-driver/driver.commit=${COMMIT} -X kamatera/shell-script-csi-driver/driver.gitTreeState=${GIT_TREE_STATE}"
CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" &&\
docker build -t kamatera/shkm-csi-plugin:dev .
```

To force update of existing deployment, delete existing objects first:

```
kubectl -n default delete pod test-shkm
kubectl -n default delete pvc test-shkm
kubectl delete pv test-shkm
kubectl -n kube-system delete daemonset csi-shkm-node
```

Deploy:

```
kubectl apply -f k8s/
```

Create test PV, PVC and Pod

```
echo '
apiVersion: v1
kind: PersistentVolume
metadata:
  name: test-shkm
spec:
  capacity:
    storage: 100G
  accessModes:
    - ReadWriteMany
  csi:
    driver: shbs.csi.kamatera.com
    volumeAttributes:
      mountScript: |
        echo mount VOLUME_ID=$VOLUME_ID TARGET_PATH=$TARGET_PATH &&\
        echo hello $VOLUME_ID > "${TARGET_PATH}/test.txt" &&\
        echo mountScript complete
      unmountScript: |
        echo mount VOLUME_ID=$VOLUME_ID TARGET_PATH=$TARGET_PATH &&\
        rm "${TARGET_PATH}/test.txt" &&\
        echo unmountScript complete
    volumeHandle: my-test-volume
' | kubectl apply -f - &&\
echo '
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-shkm
  namespace: default
spec:
  volumeName: test-shkm
  storageClassName: ""
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 100G
' | kubectl apply -f - &&\
echo '
apiVersion: v1
kind: Pod
metadata:
  name: test-shkm
  namespace: default
spec:
  containers:
  - name: alpine
    image: alpine
    command: ["sleep", "86400"]
    volumeMounts:
      - mountPath: /test
        name: test
  volumes:
    - name: test
      persistentVolumeClaim:
        claimName: test-shkm
' | kubectl apply -f -
```

Check pod

```
kubectl -n default get pod test-shkm
kubectl -n default exec test-shkm -- cat /test/test.txt
```

Check node plugin log

```
POD=`kubectl -n kube-system get pods | grep csi-shkm-node- | cut -d " " -f 1`
kubectl -n kube-system logs -c csi-shkm-plugin $POD
```

Check host path mount, get target_path from the node plugin log

```
minikube ssh -- ls -lah /var/lib/kubelet/pods/f7c21751-4a61-46b7-b2a5-c4af14c7945f/volumes/kubernetes.io~csi/test-shkm/mount
```

Delete pod

```
kubectl -n default delete pod test-shkm
```

check node plugin logs, check host path mount (should not exist)
