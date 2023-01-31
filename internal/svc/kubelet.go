package svc

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s-char-device-plugin/api"
	"k8s-char-device-plugin/internal/config"
)

const (
	kubeletSocket = "/var/lib/kubelet/device-plugins/kubelet.sock"
)

type KubeletClient struct {
	ctx                context.Context
	cancel             context.CancelFunc
	config             *config.Config
	conn               *grpc.ClientConn
	registrationClient api.RegistrationClient
	logger             logrus.FieldLogger
}

func NewKubeletClient(config *config.Config) *KubeletClient {
	cli := &KubeletClient{
		config: config,
		logger: logrus.WithField("module", "KubeletClient"),
	}
	cli.ctx, cli.cancel = context.WithCancel(context.Background())
	return cli
}

func (c *KubeletClient) Connect() error {
	var err error
	c.conn, err = grpc.DialContext(c.ctx, fmt.Sprintf("unix://%s", kubeletSocket), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	c.registrationClient = api.NewRegistrationClient(c.conn)
	return nil
}

func (c *KubeletClient) Register() error {
	names := make(map[string]struct{})
	for _, device := range c.config.Devices {
		names[device.Name] = struct{}{}
	}
	for name, _ := range names {
		_, err := c.registrationClient.Register(c.ctx, &api.RegisterRequest{
			Version:      "v1beta1",
			Endpoint:     c.config.Endpoint,
			ResourceName: fmt.Sprintf("%s/%s", c.config.ResourcePrefix, name),
			Options:      nil,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *KubeletClient) Start() {
	err := c.Connect()
	if err != nil {
		c.logger.Fatalf("Failed to connect to kubelet %v", err)
	}
	c.logger.Infof("Run initial kubelet register")
	err = c.Register()
	if err != nil {
		c.logger.Errorf("Failed to run initial kubelet register %v", err)
	}
	for {
		select {
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *KubeletClient) Stop() {
	c.cancel()
	_ = c.conn.Close()
}
