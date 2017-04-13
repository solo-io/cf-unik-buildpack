package runner

import (
	"github.com/cf-unik/unik/pkg/client"
	"github.com/cf-unik/cf-unik-buildpack/util"
	"github.com/cf-unik/pkg/errors"
	"github.com/pborman/uuid"
	"github.com/Sirupsen/logrus"
	"os"
	"strings"
	"strconv"
	"bufio"
	"fmt"
	"time"
	"net/url"
	"net/http"
	"net/http/httputil"
	"github.com/go-martini/martini"
	"syscall"
	"os/signal"
)

var port = os.Getenv("PORT")

func RunUnikernel(host string) error {
	imageName, err := util.GetAppName()
	if err != nil {
		return errors.New("could not get app name", err)
	}
	instanceName := imageName+"-"+uuid.New()
	var mountPointsToVols map[string]string
	env := make(map[string]string)
	for _, pair := range os.Environ() {
		split := strings.Split(pair, "=")
		env[split[0]] = split[1]
	}
	mem, err :=util.GetAppMem()
	if err != nil {
		return errors.New("getting app memory", err)
	}
	if memOvr := os.Getenv("MEM"); memOvr != "" {
		logrus.Info("overriding CF memory with mem %s", memOvr)
		mem, err = strconv.Atoi(memOvr)
		if err != nil {
			return errors.New("converting "+memOvr+" to int", err)
		}
	}
	logrus.WithFields(logrus.Fields{
		"host": host,
		"imageName": imageName,
		"instanceName": instanceName,
		"env": env,
		"mem": mem,
	}).Printf("running unikernel instance %v", instanceName)
	instance, err := client.UnikClient(host).Instances().Run(instanceName, imageName, mountPointsToVols, env, mem, false, false)
	if err != nil {
		return errors.New("running image failed: %v", err)
	}
	logrus.WithField("instance", instance).Info("successfully created instance")
	//if we error or terminate for any reason, always delete the instance
	defer client.UnikClient(host).Instances().Delete(instance.Id, true)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGHUP)
	go func(){
		for sig := range sigChan {
			// sig is a ^C, handle it
			logrus.Warnf(sig.String()+" detected, terminating instance")
			client.UnikClient(host).Instances().Delete(instance.Id, true)
			os.Exit(130)
		}
	}()

	//start tcp proxy
	logrus.WithField("instance", instance).Info("starting tcp proxy to instance")
	instanceIp, err := getInstanceIp(host, instance.Id)
	if err != nil {
		return errors.New("getting instance ip", err)
	}
	go startHttpProxy(instanceIp)

	//tail logs
	logrus.Info("tailing logs of instance until it dies or i do!")
	r, err := client.UnikClient(host).Instances().AttachLogs(instance.Id, true)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		if line != "\n" {
			fmt.Printf("%v", line)
		}
		if err != nil {
			return errors.New("failed to read from instance connection", err)
		}
	}
	return errors.New("instance terminated externally", nil)
}

func getInstanceIp(host, instanceId string) (string, error) {
	var ip string
	if err := retry(30, time.Second, func() error{
		instance, err := client.UnikClient(host).Instances().Get(instanceId)
		if err != nil {
			return err
		}
		if len(instance.IpAddress) > 0 {
			ip = instance.IpAddress
			return nil
		}
		logrus.Warnf("waiting for instance to get an ip... %v", instance)
		return errors.New("instance never returned an ip", nil)
	}); err != nil {
		return "", errors.New("getting instance from unik daemon", err)
	}
	return ip, nil
}

func startHttpProxy(targetIp string) {
	m := martini.Classic()
	m.Router.NotFound(func(res http.ResponseWriter, req *http.Request) {
		logrus.Infof("requested path: %v", req.URL.Path)
		if err := redirectServiceRequest(targetIp, res, req); err != nil {
			fmt.Fprintf(res, "failed to redirect request %v: %v", req.URL, err)
		}
		return
	})
	m.RunOnAddr(":"+port)
}

func redirectServiceRequest(targetIp string, res http.ResponseWriter, req *http.Request) error {
	addr := "http://" + targetIp
	if staticPort := os.Getenv("STATIC_PORT"); staticPort != "" {
		port = staticPort
	}
	addr += ":"+port

	u, err := url.Parse(addr)
	if err != nil {
		return errors.New("failed to parse ip addr "+addr, err)
	}
	httputil.NewSingleHostReverseProxy(u).ServeHTTP(res, req)
	return nil
}

func retry(retries int, sleep time.Duration, action func() error) error {
	if err := action(); err != nil {
		logrus.WithError(err).Warnf("retrying... %v", retries)
		if retries < 1 {
			return err
		}
		time.Sleep(sleep)
		return retry(retries-1, sleep, action)
	}
	return nil
}
