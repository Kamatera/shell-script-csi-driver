# Simple shell-script based CSI driver

This is a simple [Container Storage Interface](https://github.com/container-storage-interface/spec/blob/master/spec.md) driver which is based on mount / unmount shell scripts defined in the volume context.

**Warning** driver may be insecure, depending on usage, as it depends on running shell scripts in a priveleged daemonset running on all cluster nodes.

## How it works

The driver implements the node plugin interface. It runs as a daemon on all cluster nodes and handles requests to mount/unmount volumes from the node. The request context must contain `mountScript` and `unmountScript` attributes containing shell scripts to handle the mount / unmount.

## Using with Kubernetes

### Deployment

Connect to the relevant Kubernetes cluster you want to deploy to

Deploy using the following command:

```
curl -sL https://github.com/Kamatera/shell-script-csi-driver/raw/latest/k8s/releases/shell-script-csi-driver-0.0.1.yaml \
    | kubectl apply -f -
```

The main workload is deployed as a system critical DaemonSet in `kube-system` namespace with a privileged SYS_ADMIN container on the host network with bidirectional mount propagation - so that you can mount volumes inside the container that will persist to the hosting node.

The mount / unmount scripts run from within this container, so you may need to add additional tools to the driver image, depending on the shell scripts you will use.

To add additional tools, build an image based on the shell-script-csi-driver image, for example, to add bash:

```
FROM ghcr.io/kamatera/shell-script-csi-driver:latest
RUN apk --update add bash
```

Publish the image to your Docker repository, and patch the DaemonSet:

```
NEW_IMAGE_NAME="my-repo/my-image:latest"
kubectl -n kube-system patch daemonset csi-shkm-node -p '{"spec":{"template":{"spec":{"containers":[{"name":"csi-shkm-plugin","image":"'${NEW_IMAGE_NAME}'"}]}}}}'
```

### Usage

You can create PersistentVolume objects using this driver

The volumes must have the following attributes:

* `volumeHandle`: Unique name, with characters which are valid for file names
* `volumeAttributes.mountScript`: Shell script used to mount the volume to a node
* `volumeAttributes.unmountScript`: Shell script used to unmount the volume from a node

The mount/unmount scripts will run each time a pod which contains the volume is created/deleted.
 
The scripts can use the following environment variables:

* `VOLUME_ID`: The unique name from the `volumeHandle` attribute in the persistent volume
* `TARGET_PATH`: The path which the volume should be mounted to

Example PersistentVolume object:

```
apiVersion: v1
kind: PersistentVolume
metadata:
  name: test-shkm
spec:
  capacity:
    # storage size is not used, but must be present in the spec
    storage: 100G
  accessModes:
    # access modes are not used, but must be present in the spec
    - ReadWriteMany
  csi:
    driver: shbs.csi.kamatera.com
    volumeHandle: my-test-volume
    volumeAttributes:
      mountScript: |
        echo mount VOLUME_ID=$VOLUME_ID TARGET_PATH=$TARGET_PATH &&\
        echo hello $VOLUME_ID > "${TARGET_PATH}/test.txt" &&\
        echo mountScript complete
      unmountScript: |
        echo mount VOLUME_ID=$VOLUME_ID TARGET_PATH=$TARGET_PATH &&\
        rm "${TARGET_PATH}/test.txt" &&\
        echo unmountScript complete
```

Example PersistentVolumeClaim and Pod objects using this volume:

```
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
```

```
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
```

### Debugging

Get the daemonset pod name running on each node:

```
kubectl -n kube-system get pods -l app=csi-shkm-node -o custom-columns=pod:.metadata.name,node:.spec.nodeName
```

Get logs:

```
kubectl -n kube-system logs -c csi-shkm-plugin POD_NAME
```
