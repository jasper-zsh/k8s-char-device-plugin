package util

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Service interface {
	Start()
	Stop()
}

type ServiceGroup struct {
	services []Service
}

func NewServiceGroup() *ServiceGroup {
	return &ServiceGroup{
		services: make([]Service, 0),
	}
}

func (g *ServiceGroup) AddService(svc Service) {
	g.services = append(g.services, svc)
}

func (g *ServiceGroup) Start() {
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill, syscall.SIGTERM)
	wg := sync.WaitGroup{}
	for _, svc := range g.services {
		svc := svc
		wg.Add(1)
		go func() {
			svc.Start()
			wg.Done()
		}()
	}
	go func() {
		defer func() {
			for _, svc := range g.services {
				svc.Stop()
			}
		}()
		for {
			select {
			case <-sig:
				return
			}
		}
	}()
	wg.Wait()
}
