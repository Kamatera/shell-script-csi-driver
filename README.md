# Simple shell-script based CSI driver

This is a simple [Container Storage Interface](https://github.com/container-storage-interface/spec/blob/master/spec.md) driver which is based on mount / unmount shell scripts defined in the volume context.

**Warning** driver may be insecure, depending on usage, as it depends on running shell scripts on the node, based on CSI driver volume context. 

## Using with Kubernetes

### Deployment

Connect to the relevant Kubernetes cluster you want to deploy to

Deploy using the following command:

```
curl -sL https://github.com/Kamatera/shell-script-csi-driver/raw/latest/k8s/releases/shell-script-csi-driver-0.0.1.yaml \
    | kubectl apply -f -
```

You may need to add additional tools to the driver image, depending on the shell scripts you will use

Build an image based on the shell-script-csi-driver docker, for example, to add bash:

```
FROM kamatera/shkm-csi-plugin:0.0.1
RUN apk --update add bash
```

Publish the image, and patch the DaemonSet:

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
