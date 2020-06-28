package main

import (
	"flag"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
	"kube-webhook/src/common"
	"kube-webhook/src/config"
	"kube-webhook/src/webhook"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var webHookParam webhook.WebHookServerParameters

func main() {
	// parse parameters
	flag.Parse()
	defer glog.Flush()

	portConfig := config.PortConfig{}
	portConfig.Init(webHookParam.PortConfigFile)

	// 定期更新端口配置,实际间隔时长为：间隔时间 + 执行时间
	go wait.Forever(portConfig.RefreshPortConfig, 10*time.Second)

	// 定时记录内存使用情况
	sys := common.SYS{}
	go wait.Forever(sys.LogMemoryInfo, 10*time.Minute)

	// init webhook api
	ws, err := webhook.NewWebhookServer(webHookParam)
	if err != nil {
		panic(err)
	}

	// start webhook server in new routine
	go ws.Start()
	glog.Info("Server started")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	ws.Stop()
}

func init() {
	// read parameters
	flag.IntVar(&webHookParam.Port, "port", 443, "The port of webhook server to listen.")
	flag.StringVar(&webHookParam.CertFile, "tlsCertPath", "/etc/webhook/certs/cert.pem", "The path of tls cert")
	flag.StringVar(&webHookParam.KeyFile, "tlsKeyPath", "/etc/webhook/certs/key.pem", "The path of tls key")
	flag.StringVar(&webHookParam.PortConfigFile, "portConfigFile", "/etc/webhook/conf/ports.conf", "Nodeport config file path")
	// 需要忽略的namespace，以逗号分隔，不管是否配置总是默认添加kube-public,kube-system两个namespace
	flag.StringVar(&webHookParam.IgnoredNamespaces, "ignoredNS", "kube-public,kube-system","ignored namespaces,default kube-public,kube-system")
}