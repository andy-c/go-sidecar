/**
  @description:go-sidecar
  @author: Angels lose their hair
  @date: 2021/5/12
  @version:v1
**/
package registercenter

import (
	"context"
	"encoding/json"
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/levigross/grequests"
	"github.com/sirupsen/logrus"
	"go-sidecar/config"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	PullTimeout=4000
	EurekaApiPrefix="/eureka/apps/"
	HeartbeatTimeout=10000
	PullAppTimeout=5000
	ConsulApiPrefix="/v1/agent/service/"
	ConsulHost="http://127.0.0.1:8500"
)

type RegisterCenter interface {
	HeartBeat()
	Register()
	Deregister()
	Instances()
}

var Singleton *Eureka
var allApps = &Applications{}
var ConsulSingleton *Consul

type Eureka struct{
	ctx context.Context
	Status bool //eureka register status
	wait sync.WaitGroup
	mu sync.RWMutex
	instance *Instance
}

func NewEureka(mainCtx context.Context){
	Singleton=&Eureka{
		ctx:mainCtx,
		Status: false,
	}
	Singleton.setInstance()
	go Singleton.run()
	go Singleton.handleContext()

}

func (r *Eureka) setInstance(){
	port,err:=strconv.Atoi(config.Config.Eureka.AppPort)
	if err!=nil{
		logrus.Error("app port useless")
		return
	}
	r.instance=&Instance{
		InstanceId: config.Config.Eureka.AppAddr+":"+config.Config.Eureka.AppName+":"+config.Config.Eureka.AppPort,
		HostName: config.Config.Eureka.HostName,
		App:config.Config.Eureka.AppName,
		IpAddr: config.Config.Eureka.AppAddr,
		Status: "UP",
		OverriddenStatus:"UNKNOWN",
		Port: &Port{
			Port:port ,
			Enabled: "true",
		},
		SecurePort: &SecurePort{
			Port:443,
			Enabled: "false",
		},
		CountryId: 1,
		DataCenterInfo: &DataCenterInfo{
			Class: "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo",
			Name: "MyOwn",
		},
		LeaseInfo: &LeaseInfo{
			RenewalIntervalInSecs: 15,
			DurationInSecs: 90,
			RegistrationTimestamp: time.Now().UnixNano() / 10e5,
			LastRenewalTimestamp: 0,
			EvictionTimestamp: 0,
			ServiceUpTimestamp: time.Now().UnixNano() / 10e5,
		},
		Metadata: &Metadata{
			Class: "",
		},
		HomePageUrl: "http://"+config.Config.Eureka.AppAddr+":"+config.Config.Eureka.AppPort+"/"+config.Config.Eureka.HomePageUri,
		StatusPageUrl: "http://"+config.Config.Eureka.AppAddr+":"+config.Config.Eureka.AppPort+"/"+config.Config.Eureka.StatusPageUri,
		HealthCheckUrl: "http://"+config.Config.Eureka.AppAddr+":"+config.Config.Eureka.AppPort+"/"+config.Config.Eureka.HealthCheckUri,
		VipAddress: "http://"+config.Config.Eureka.AppAddr,
		SecureVipAddress: "http://"+config.Config.Eureka.AppAddr,
		IsCoordinatingDiscoveryServer: "false",
		LastUpdatedTimestamp: strconv.FormatInt(time.Now().UnixNano()/10e5,10),
		LastDirtyTimestamp: strconv.FormatInt(time.Now().UnixNano()/10e5,10),
	}
}

func (r *Eureka) GetInstanceInfo() *Instance{
	return r.instance
}

func (r *Eureka) handleContext(){
    config.WaitGroup.Add(1)
    defer config.WaitGroup.Done()
	for{
		select {
		    case <- r.ctx.Done():
				logrus.Info("eureka goroutine done")
		    	r.Deregister()
		    	return
		}
	}
}

func (r *Eureka) run(){
	//we need register firstly
	go r.Register()
	go r.HeartBeat()
	go r.ServiceDiscovery()
}

func (r *Eureka) HeartBeat(){
   config.WaitGroup.Add(1)
   defer config.WaitGroup.Done()
   ticker:=time.NewTicker(time.Duration(HeartbeatTimeout)*time.Millisecond)
   defer ticker.Stop()
   for{
	   select {
   	      case <-r.ctx.Done():
   	      	logrus.Info("eureka done")
   	      	return
   	      case <-ticker.C:
   	         r.uploadHeartPacket()
	  }
   }
}

func (r *Eureka) uploadHeartPacket(){
	if !r.Status {
		return
	}
	var response *grequests.Response
	var err error
	//random a eureka client
	eurekaAdders :=strings.Split(config.Config.Eureka.EurekServerAddr,",")
	rand.Seed(time.Now().Unix())
    index:=rand.Intn(len(eurekaAdders))
    hitAddr:= eurekaAdders[index]
    //heart beat query
    heartBeatQuery:=make(map[string]string)
    heartBeatQuery["Status"] = r.getProxyAppStatus()
    heartBeatQuery["lastDirtyTimestamp"] = r.instance.LastDirtyTimestamp
	options:=&grequests.RequestOptions{
		TLSHandshakeTimeout: time.Duration(PullTimeout)*time.Millisecond,
		DialTimeout:         time.Duration(PullTimeout)*time.Millisecond,
		RequestTimeout:      time.Duration(PullTimeout)*time.Millisecond,
		Context: r.ctx,
		Params: heartBeatQuery,
		Headers: r.buildEurekaRequestHeader(),
	}
	defer func() {
		if response==nil{
			return
		}
		err:=response.Close()
		if err!=nil{
			logrus.Error("eureka heartbeat close response failed,error is "+err.Error())
		}
	}()
    instanceId:=config.Config.Eureka.AppAddr+":"+config.Config.Eureka.AppName+":"+config.Config.Eureka.AppPort
	response, err = grequests.Put(hitAddr+EurekaApiPrefix+config.Config.Eureka.AppName+"/"+instanceId, options)
	if err!=nil{
		logrus.Error("eureka heartbeat failed ,error is "+err.Error())
	}

	if response == nil {
		logrus.Error("eureka heartbeat response is nil")
		return
	}
	if response.StatusCode==404 {
		logrus.Error("eureka heartbeat status 404,response error is "+response.Error.Error())
		return
	}
    if response.StatusCode !=200{
    	logrus.Errorf("eureka heartbeat status code is %d",response.StatusCode)
    	return
	}
	r.instance.LeaseInfo.LastRenewalTimestamp = time.Now().UnixNano() / 10e5
}

func (r *Eureka) getProxyAppStatus() string{
	var response *grequests.Response
	var err error
	status:="UP"

	options:=&grequests.RequestOptions{
		TLSHandshakeTimeout: time.Duration(PullTimeout)*time.Millisecond,
		DialTimeout:         time.Duration(PullTimeout)*time.Millisecond,
		RequestTimeout:      time.Duration(PullTimeout)*time.Millisecond,
		Context: r.ctx,
		Headers: r.buildHttpHeader(),
	}

	response,err= grequests.Get("http://"+config.Config.Eureka.AppAddr+":"+config.Config.Eureka.AppPort+"/"+config.Config.Eureka.HealthCheckUri,options)
	if err!=nil{
		logrus.Error("eureka get proxy app status is down")
		status = "DOWN"
	}
	if response.StatusCode!=200{
		status="DOWN"
		logrus.Error("eureka get proxy app status is down")
	}
	return status
}

func (r *Eureka) Register(){
   //we need register all eureka instance
   eurekas:=strings.Split(config.Config.Eureka.EurekServerAddr,",")
   responseSlice:=make([]*grequests.Response,len(eurekas))
   var err error
   defer func() {
   	   if responseSlice == nil || len(responseSlice) == 0 {
   	   	  return
	   }

	   for _,resp:= range responseSlice{
	   	   if r==nil{
	   	   	 continue
		   }
		   err:=resp.Close()
		   if resp.StatusCode == 204 || resp.StatusCode == 200 {
		   	   r.Status = true
		   }
		   if err!=nil{
		   	  logrus.Error("eureka register failed ,error is "+err.Error())
		   }
	   }
   }()

   //send request
   for key,eurekaAddr:=range eurekas {
	   options:=&grequests.RequestOptions{
		   TLSHandshakeTimeout: time.Duration(PullTimeout)*time.Millisecond,
		   DialTimeout:         time.Duration(PullTimeout)*time.Millisecond,
		   RequestTimeout:      time.Duration(PullTimeout)*time.Millisecond,
		   Context:             r.ctx,
		   JSON:                map[string]interface{}{"instance":r.instance},
	   }
	   responseSlice[key],err =grequests.Post(eurekaAddr+EurekaApiPrefix+config.Config.Eureka.AppName,options)
	   if err!=nil{
	   	   logrus.Error("eureka register failed error is "+err.Error())
	   }else{
	   	  if responseSlice[key].StatusCode == 204 {
	   	  	  logrus.Info("eureka has been registered,register adder is "+eurekaAddr)
		  }
		  if responseSlice[key] == nil {
		  	  logrus.Info("eureka register status is unknown,register adder is "+eurekaAddr)
		  }
		  if responseSlice[key]!=nil && responseSlice[key].StatusCode == 200 {
		  	  logrus.Infof("eureka register success ,addr is %s",eurekaAddr)
		  }
	   }
   }
}


func (r *Eureka) Deregister(){
   var response *grequests.Response
   var err error
   status:=make(map[string]int)
   i:=0
   var loop = false

   if !r.Status {
   	  return
   }
   defer func() {
   	  if response != nil {
   	  	  response.Close()
	  }
   }()

   for i<10 {
   	   eurekaAdders := strings.Split(config.Config.Eureka.EurekServerAddr,",")
   	   for _,adder :=range eurekaAdders {
   	   	   if status[adder] == 200 || status[adder] == 404 {
   	   	   	  continue
		   }
   	   	   options:=&grequests.RequestOptions{
			   TLSHandshakeTimeout: time.Duration(PullTimeout)*time.Millisecond,
			   DialTimeout:         time.Duration(PullTimeout)*time.Millisecond,
			   RequestTimeout:      time.Duration(PullTimeout)*time.Millisecond,
			   Headers: r.buildHttpHeader(),
		   }

		   response,err = grequests.Delete(adder+EurekaApiPrefix+config.Config.Eureka.AppName+"/"+config.Config.Eureka.AppAddr+":"+config.Config.Eureka.AppName+":"+config.Config.Eureka.AppPort,options)
		   if err !=nil{
		   	  logrus.Errorf("eureka deregister failed ,adder is %s",adder)
		   	  continue
		   }

		   if response == nil {
		   	   logrus.Error("eureka deregister failed ,response is empty,deregister adder is "+adder)
		   	   continue
		   }

		   if response.StatusCode!=200 && response.StatusCode!=404{
			   logrus.Errorf("eureka deregister failed ,response code is %d,adder is %s ",response.StatusCode,adder)
			   continue
		   }
		   status[adder] = response.StatusCode
	   }

	   for _,v:=range status{
	   	  if v!=200 && v!=404 {
	   	  	 loop = true
		  }else{
		  	 loop = false
		  }
	   }

	   if loop {
	   	  i++
	   }else{
	   	  break
	   }
   }
}
func (r *Eureka) ServiceDiscovery(){
   config.WaitGroup.Add(1)
   defer config.WaitGroup.Done()
   ticker:=time.NewTicker(time.Duration(PullAppTimeout)*time.Millisecond)
   defer ticker.Stop()

   r.pullProxyApps()

   for{
	   select {
   	      case <-r.ctx.Done():
   	      	logrus.Info("eureka service discovery done")
   	      	return
   	      case <-ticker.C:
   	        r.pullProxyApps()
	  }
   }

}
func (r *Eureka) pullProxyApps(){
	var response *grequests.Response
	var err error

	//firstly we need check apps version
	rand.Seed(time.Now().Unix())
	eurekaAdders:= strings.Split(config.Config.Eureka.EurekServerAddr, ",")
    index:=rand.Intn(len(eurekaAdders))
    randEurekaAdder:=eurekaAdders[index]

    defer func() {
    	if response!=nil {
    		response.Close()
		}
	}()
	options:=&grequests.RequestOptions{
		TLSHandshakeTimeout: time.Duration(PullTimeout)*time.Millisecond,
		DialTimeout:         time.Duration(PullTimeout)*time.Millisecond,
		RequestTimeout:      time.Duration(PullTimeout)*time.Millisecond,
		Context: r.ctx,
		Headers: r.buildHttpHeader(),
	}
	response,err = grequests.Get(randEurekaAdder+EurekaApiPrefix+"delta",options)
	if err != nil {
		logrus.Error("eureka check delta failed ,error is "+err.Error())
		return
	}

	if response == nil {
		logrus.Error("eureka delta response failed")
		return
	}
	if response.StatusCode!=200{
		logrus.Error("eureka delta status code is "+string(response.StatusCode))
		return
	}
	eurekaApps:=&EurekaApps{}
	err = json.Unmarshal(response.Bytes(),eurekaApps)
	if err !=nil{
		logrus.Error("eureka pull apps json decode failed,error is "+err.Error())
		return
	}
	apps:= &eurekaApps.Apps

	if  apps.VersionDelta == allApps.VersionDelta {
		logrus.Info("eureka pull apps ,dont need update")
		return
	}
	//we need pull all apps
	allapps,err:=grequests.Get(randEurekaAdder+EurekaApiPrefix,&grequests.RequestOptions{
		TLSHandshakeTimeout: time.Duration(PullTimeout)*time.Millisecond,
		DialTimeout:         time.Duration(PullTimeout)*time.Millisecond,
		RequestTimeout:      time.Duration(PullTimeout)*time.Millisecond,
		Context: r.ctx,
		Headers: r.buildHttpHeader(),
	})
	//fix memory leak
	defer func() {
		if allapps!=nil {
			allapps.Close()
		}
	}()
	if err !=nil {
		logrus.Error("eureka pull apps failed ,error is "+err.Error())
		return
	}
	if allapps == nil {
		logrus.Error("eureka pull apps failed,response body is empty")
		return
	}
	if allapps.StatusCode!=200{
		logrus.Error("eureka pull apps failed,response code is "+string(allapps.StatusCode))
		return
	}
	eurekaApps= &EurekaApps{}
	err = json.Unmarshal(allapps.Bytes(), eurekaApps)
	if err != nil {
		logrus.Error("eureka pull apps json decode failed,error is "+err.Error())
		return
	}

	apps = &eurekaApps.Apps
	if apps.VersionDelta == "" {
		logrus.Error("eureka pull apps version delta is empty")
		return
	}

	apps.Index = make(map[string]int)
	for i, app := range apps.Application {
		apps.Index[app.Name] = i
	}

	r.updateApps(apps, apps.VersionDelta)
}

//update apps info
func (r *Eureka) updateApps(apps *Applications, appsVersion string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	allApps = apps
	allApps.VersionDelta = appsVersion
}

func (r *Eureka) buildEurekaRequestHeader() map[string]string{
	defaultHeader:=make(map[string]string)
	defaultHeader["Accept-Encoding"] = "gzip"
	defaultHeader["DiscoveryIdentity-Name"] = "DefaultClient"
	defaultHeader["DiscoveryIdentity-Version"] = config.Config.Eureka.AppAddr+":"+config.Config.Eureka.AppName+":"+config.Config.Eureka.AppPort
	defaultHeader["Connection"] = "Keep-Alive"
	defaultHeader["Accept"] = "application/json"
	defaultHeader["Content-Type'"] = "application/json"
	return defaultHeader
}

func (r *Eureka) buildHttpHeader() map[string]string{
	defaultHeader:=make(map[string]string)
	defaultHeader["Accept"] = "application/json"
	defaultHeader["Content-Type'"] = "application/json"
	defaultHeader["User-Agent"] = config.Config.Eureka.AppName
	return defaultHeader
}


func (r *Eureka) Random(sname string) *Instance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var index int
	var instance *Instance

	if i,ok:=allApps.Index[strings.ToUpper(sname)];!ok{
		index = i
		return nil
	}

	if len(allApps.Application[index].Instance) <1 {
		return nil
	}

	if len(allApps.Application[index].Instance) == 1 {
		instance = &allApps.Application[index].Instance[0]
	}else{ //we need random a service
		rand.Seed(time.Now().Unix())
		sIndex:=rand.Intn(len(allApps.Application[index].Instance))
		randS:=allApps.Application[index].Instance[sIndex]
		if randS.Status != "UP" {
			for _,v:=range allApps.Application[index].Instance {
				if v.Status == "UP"{
					instance = &v
					break
				}
			}
		}
	}

	return instance
}

func (r *Eureka) GetInstances() []Application{
	return allApps.Application
}


type Consul struct{
	consulAgent *consulapi.Client
	ctx context.Context
}

func NewConsul(mainCtx context.Context){
	ConsulSingleton=&Consul{
		ctx:mainCtx,
	}
	var err error
	ConsulSingleton.consulAgent,err = consulapi.NewClient(consulapi.DefaultConfig())
    if err!=nil {
		logrus.Error("init consul error,error is " + err.Error())
		os.Exit(1)
	}
	go ConsulSingleton.Register()
    go ConsulSingleton.handleContext()
}

func (c *Consul) handleContext(){
	config.WaitGroup.Add(1)
	defer config.WaitGroup.Done()
	for{
		select {
		    case <-c.ctx.Done():
		    	logrus.Error("consul deregister")
		    	c.Deregister()
		    	return
		}
	}
}

func (c *Consul) Register(){
    var err error
    instance:=&consulapi.AgentServiceRegistration{
    	ID:config.Config.Consul.AppName,
    	Name:config.Config.Consul.AppName,
    	Port:8090,
    	Address:config.Config.Consul.AppAddr,
    	Tags: []string{"instance_node"},
    	Check: &consulapi.AgentServiceCheck{
    		HTTP: fmt.Sprintf("http://%s:%d/%s",config.Config.Server.Host,config.Config.Server.Port,config.Config.Consul.HealthCheckUri),
    		Timeout: "5s",
    		Interval: "5s",
    		DeregisterCriticalServiceAfter: "60s",//check failed,then wait 60 seconds to remove instance
		},
	}
	err = c.consulAgent.Agent().ServiceRegister(instance)
	if err!=nil{
		logrus.Error("register instance failed,error is "+err.Error())
	}
}

func (c *Consul) Deregister(){
	var i = 0
	for i< 10{
		err:=c.consulAgent.Agent().ServiceDeregister(config.Config.Consul.AppName)
		if err !=nil {
			logrus.Error("deregister failed ,error is "+err.Error())
			i++
		}else{
			break
		}
	}
}

func (c *Consul) Random(sName string) *consulapi.AgentService{
	options:=&consulapi.QueryOptions{
		WaitTime: time.Duration(4000)*time.Millisecond,
	}
	instance,_,err :=c.consulAgent.Agent().Service(sName,options)
	if err != nil {
		logrus.Error("fetch consul service failed,error is "+err.Error())
	}
	return instance
}

