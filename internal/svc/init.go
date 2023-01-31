package svc

import (
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s-char-device-plugin/api"
	"k8s-char-device-plugin/internal/config"
	"net"
	"path"
	"time"
)

const (
	DevicePluginsSocketDir = "/var/lib/kubelet/device-plugins"
)

type Server struct {
	internalCtx         context.Context
	internalCancel      context.CancelFunc
	ctx                 context.Context
	cancel              context.CancelFunc
	config              *config.Config
	grpcServer          *grpc.Server
	devicePluginService *DevicePluginService
	kubeletClient       *KubeletClient
	pluginSocketWatcher *fsnotify.Watcher
	logger              logrus.FieldLogger
}

func NewServer(config *config.Config) *Server {
	s := &Server{
		config:              config,
		devicePluginService: NewDevicePluginServer(config),
		logger:              logrus.WithField("module", "Server"),
	}
	return s
}

func (s *Server) serve() {
	l, err := net.Listen("unix", path.Join(DevicePluginsSocketDir, s.config.Endpoint))
	if err != nil {
		s.logger.Fatalf("Failed to start device plugin server: %v", err)
	}
	s.grpcServer = grpc.NewServer()
	api.RegisterDevicePluginServer(s.grpcServer, s.devicePluginService)
	err = s.grpcServer.Serve(l)
	if err != nil {
		s.logger.Errorf("Error when serving device plugin server: %v", err)
	}
}

func (s *Server) Start() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			s.start()
		}
	}
}

func (s *Server) stop() {
	s.internalCancel()
	s.grpcServer.GracefulStop()
	s.kubeletClient.Stop()
}

func (s *Server) start() {
	var err error
	s.internalCtx, s.internalCancel = context.WithCancel(s.ctx)
	go func() {
		s.serve()
	}()
	s.pluginSocketWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		s.logger.Fatalf("Failed to create watcher %v", err)
	}
	defer s.pluginSocketWatcher.Close()
	err = s.pluginSocketWatcher.Add(DevicePluginsSocketDir)
	if err != nil {
		s.logger.Fatalf("Failed to watch plugin socket %v", err)
	}
	s.kubeletClient = NewKubeletClient(s.config)
	go s.kubeletClient.Start()
	for {
		select {
		case <-s.internalCtx.Done():
			return
		case event, ok := <-s.pluginSocketWatcher.Events:
			if !ok {
				return
			}
			if event.Name == fmt.Sprintf("%s/%s", DevicePluginsSocketDir, s.config.Endpoint) && event.Has(fsnotify.Remove) {
				s.stop()
				s.logger.Infof("Kubernetes restarted, wait 5 seconds to re-register")
				time.Sleep(5 * time.Second)
			}
		case err, ok := <-s.pluginSocketWatcher.Errors:
			if !ok {
				return
			}
			s.logger.Errorf("Error when watching plugin socket %v", err)
		}
	}
}

func (s *Server) Stop() {
	s.cancel()
	s.stop()
}
