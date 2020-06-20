package driver

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (d *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeStageVolume not supported")
}

// NodeUnstageVolume unstages the volume from the staging path
func (d *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeUnstageVolume not supported")
}

// NodePublishVolume mounts the volume mounted to the staging path to the target path
func (d *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume Volume ID must be provided")
	}

	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume Target Path must be provided")
	}

	mountScript, unmountScript := "", ""
	for key, value := range req.VolumeContext {
		if key == "mountScript" {
			mountScript = value
		} else if key == "unmountScript" {
			unmountScript = value
		}
	}
	if mountScript == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume mountScript must be provided")
	}
	if unmountScript == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume unmountScript must be provided")
	}

	log := d.log.WithFields(logrus.Fields{
		"volume_id":           req.VolumeId,
		"target_path":         req.TargetPath,
		"method":              "node_publish_volume",
	})
	log.WithField("req", req).Info("node publish volume called")

	err := ioutil.WriteFile(path.Join(d.workdir, req.VolumeId + ".unmount"), []byte(unmountScript), 0644)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}

	// create target, os.Mkdirall is noop if directory exists
	err = os.MkdirAll(req.TargetPath, 0750)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}

	mountCmd := "sh"
	mountArgs := []string{"-c", "export TARGET_PATH=\"" + req.TargetPath + "\"\nexport VOLUME_ID=\"" + req.VolumeId + "\"\n" + mountScript}

	log.WithFields(logrus.Fields{
		"cmd":  mountCmd,
		"args": mountArgs,
	}).Info("executing mount command")

	out, err := exec.Command(mountCmd, mountArgs...).CombinedOutput()
	if err != nil {
		return nil, status.Error(
			codes.Unknown,
			fmt.Sprintf(
				"mounting failed: %v cmd: '%s %s' output: %q",
				err, mountCmd, strings.Join(mountArgs, " "), string(out),
			),
		)
	}

	log.WithField("out", string(out)).Info("bind mounting the volume is finished")
	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume unmounts the volume from the target path
func (d *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume Volume ID must be provided")
	}

	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume Target Path must be provided")
	}

	log := d.log.WithFields(logrus.Fields{
		"volume_id":   req.VolumeId,
		"target_path": req.TargetPath,
		"method":      "node_unpublish_volume",
	})
	log.WithField("req", req).Info("node unpublish volume called")

	file, err := os.Open(path.Join(d.workdir, req.VolumeId + ".unmount"))
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}

	defer file.Close()
	respBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}
	unmountScript := string(respBytes)

	unmountCmd := "sh"
	unmountArgs := []string{"-c", "export TARGET_PATH=\"" + req.TargetPath + "\"\nexport VOLUME_ID=\"" + req.VolumeId + "\"\n" + unmountScript}

	out, err := exec.Command(unmountCmd, unmountArgs...).CombinedOutput()
	if err != nil {
		return nil, status.Error(
			codes.Unknown,
			fmt.Sprintf(
				"mounting failed: %v cmd: '%s %s' output: %q",
				err, unmountCmd, strings.Join(unmountArgs, " "), string(out),
			),
		)
	}

	log.WithField("out", string(out)).Info("unmounting volume is finished")
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeGetCapabilities returns the supported capabilities of the node server
func (d *Driver) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	nscaps := []*csi.NodeServiceCapability{}
	d.log.WithFields(logrus.Fields{
		"node_capabilities": nscaps,
		"method":            "node_get_capabilities",
	}).Info("node get capabilities called")
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: nscaps,
	}, nil
}

// NodeGetInfo returns the supported capabilities of the node server.
// This is used so the CO knows where to place the workload. The result of this function will be used
// by the CO in ControllerPublishVolume.
func (d *Driver) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	d.log.WithField("method", "node_get_info").Info("node get info called")
	return &csi.NodeGetInfoResponse{
		NodeId:            d.hostID,
	}, nil
}

// NodeGetVolumeStats returns the volume capacity statistics available for the
// the given volume.
func (d *Driver) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeGetVolumeStats is not supported")
}

func (d *Driver) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeExpandVolume is not supported")
}
