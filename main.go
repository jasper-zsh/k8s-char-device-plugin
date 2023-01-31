package main

import (
	"flag"
	"fmt"
	"k8s-char-device-plugin/internal/config"
	"k8s-char-device-plugin/internal/svc"
	"k8s-char-device-plugin/internal/util"
)

var configFile = flag.String("config", "/etc/k8s-char-device-plugin/config.yaml", "Path to config.yaml")

func main() {
	flag.Parse()
	c := config.MustLoadConfigFromFile(*configFile)
	fmt.Printf("%+v\n", c)

	sg := util.NewServiceGroup()
	sg.AddService(svc.NewServer(c))
	sg.Start()
}
