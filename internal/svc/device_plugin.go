package svc

import (
	context "context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"k8s-char-device-plugin/api"
	"k8s-char-device-plugin/internal/config"
	"os"
	"strings"
)

const (
	deviceIDSeprator = "#"
)

var _ api.DevicePluginServer = (*DevicePluginService)(nil)

type DevicePluginService struct {
	api.UnimplementedDevicePluginServer
	config   *config.Config
	hostname string
	logger   logrus.FieldLogger
}

func NewDevicePluginServer(config *config.Config) *DevicePluginService {
	s := &DevicePluginService{
		config: config,
		logger: logrus.WithField("module", "DevicePluginServer"),
	}
	var err error
	s.hostname, err = os.Hostname()
	if err != nil {
		s.logger.Fatalf("Failed to get hostname %v", err)
	}
	return s
}

func (d *DevicePluginService) GetDevicePluginOptions(ctx context.Context, empty *api.Empty) (*api.DevicePluginOptions, error) {
	return &api.DevicePluginOptions{}, nil
}

func (d *DevicePluginService) deviceID(devicePath string) string {
	return fmt.Sprintf("CHAR%s%s%s%s", deviceIDSeprator, d.hostname, deviceIDSeprator, devicePath)
}

func (d *DevicePluginService) buildDeviceList() []*api.Device {
	ret := make([]*api.Device, 0, len(d.config.Devices))
	for _, device := range d.config.Devices {
		_, err := os.Stat(device.Path)
		if err == nil {
			ret = append(ret, &api.Device{
				ID:     d.deviceID(device.Path),
				Health: "Healthy",
			})
		}
	}
	return ret
}

func (d *DevicePluginService) ListAndWatch(empty *api.Empty, server api.DevicePlugin_ListAndWatchServer) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	for _, device := range d.config.Devices {
		err := watcher.Add(device.Path)
		if err != nil {
			d.logger.Errorf("Failed to watch %s %v", device, err)
			continue
		}
	}
	devices := d.buildDeviceList()
	d.logger.Infof("Sending initial device list %v", devices)
	err = server.Send(&api.ListAndWatchResponse{
		Devices: devices,
	})
	if err != nil {
		d.logger.Errorf("Failed to send initial device list %v", err)
	}
	for {
		select {
		case <-server.Context().Done():
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
				devices := d.buildDeviceList()
				d.logger.Infof("Sending device list %v", devices)
				err := server.Send(&api.ListAndWatchResponse{
					Devices: devices,
				})
				if err != nil {
					d.logger.Errorf("Failed to send device list %v", err)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			d.logger.Errorf("Error when watching devices %v", err)
		}
	}
}

func (d *DevicePluginService) GetPreferredAllocation(ctx context.Context, request *api.PreferredAllocationRequest) (*api.PreferredAllocationResponse, error) {
	panic("do not need")
}

func (d *DevicePluginService) Allocate(ctx context.Context, request *api.AllocateRequest) (*api.AllocateResponse, error) {
	res := make([]*api.ContainerAllocateResponse, 0, len(request.ContainerRequests))
	for _, req := range request.ContainerRequests {
		devices := make([]*api.DeviceSpec, 0, len(req.DevicesIds))
		for _, deviceID := range req.DevicesIds {
			parts := strings.Split(deviceID, deviceIDSeprator)
			devicePath := parts[len(parts)-1]
			device := &api.DeviceSpec{
				ContainerPath: devicePath,
				HostPath:      devicePath,
				Permissions:   "rw",
			}
			d.logger.Infof("Allocating device %s %v", deviceID, device)
			devices = append(devices, device)
		}
		res = append(res, &api.ContainerAllocateResponse{
			Devices: devices,
		})
	}
	return &api.AllocateResponse{
		ContainerResponses: res,
	}, nil
}

func (d *DevicePluginService) PreStartContainer(ctx context.Context, request *api.PreStartContainerRequest) (*api.PreStartContainerResponse, error) {
	panic("do not need")
}
