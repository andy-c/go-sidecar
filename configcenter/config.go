/**
  @description:go-sidecar
  @author: Angels lose their hair
  @date: 2021/5/12
  @version:v1
**/
package configcenter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/levigross/grequests"
	"github.com/sirupsen/logrus"
	"go-sidecar/config"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"
)

const (
	PullTimeout = 6000 // for connect ,request
	HoldTimeout = 65000 //for long pull timeout
	TickTimeout = 9000 //for timer goroutine
)

type Config interface {
	//it is a timer tick goroutine as backup
	//when the listen goroutine failed
	Timer()

	//pull the config from config center
	Listen()
}

var singleton *Apollo

//crtip apollo config center
type Apollo struct{
	ctx context.Context
	notifications map[string]*Notification //we need modify the notification,so we should use the pointer
	mu sync.RWMutex
}

type Notification struct {
	NamespaceName  string  `json:"namespaceName"`
	NotificationId float64 `json:"notificationId"`
}


func NewApollo(mainCtx context.Context){
	singleton=&Apollo{
		ctx:mainCtx,
	}
	err:=singleton.initNotification()
	if err!=nil{
		logrus.Error(err.Error())
	}
	go singleton.run()
	go singleton.handleContext()

}

func (cs *Apollo) initNotification() error{
	cs.mu.Lock()
	defer cs.mu.Unlock()

	//check config center notification
	if config.Config.Apollo.Notifications == "" {
		return errors.New("notifications is empty")
	}
	cs.notifications = make(map[string]*Notification)
	for _, namespace := range strings.Split(config.Config.Apollo.Notifications, ",") {
		cs.notifications[namespace] = &Notification{}
		cs.notifications[namespace].NamespaceName = namespace
		cs.notifications[namespace].NotificationId = -1
	}
	return nil
}

//run timer and listen
func (cs *Apollo) run(){
	go cs.Timer()
	go cs.Listen()
}

//handle context
func (cs *Apollo) handleContext(){
	config.WaitGroup.Add(1)
	defer config.WaitGroup.Done()
	for{
		select {
		    case <-cs.ctx.Done():
		    	logrus.Info("apollo done")
		    	return
		}
	}
}


func (cs *Apollo) Timer(){
	config.WaitGroup.Add(1)
	defer config.WaitGroup.Done()

	ticker := time.NewTicker(time.Duration(TickTimeout) * time.Millisecond)
	defer ticker.Stop()
	for{
		select {
		   case <-cs.ctx.Done():
		   	    logrus.Info("timer goroutine done")
		   	    return
		   case <-ticker.C:
		   	    cs.handleConfiguration()
		}
	}

}

func (cs *Apollo) Listen(){
	config.WaitGroup.Add(1)
	defer config.WaitGroup.Done()
	for {
		select {
		case <-cs.ctx.Done():
			logrus.Info("  listen goroutine done")
			return
		default:
			cs.handleConfiguration()
		}
	}
}

func(cs *Apollo) handleConfiguration(){
   query:=make(map[string]string)
   query["appId"]   = config.Config.Apollo.AppId
   query["cluster"] = config.Config.Apollo.ClusterName
   notificationSlice := cs.getNotificationsSlice()
   notificationBytes,err:=json.Marshal(notificationSlice)
   if err != nil {
   	  query["notifications"] = "{}"
   }else{
   	  query["notifications"] = string(notificationBytes)
   }

   options:=&grequests.RequestOptions{
      TLSHandshakeTimeout: time.Duration(PullTimeout)*time.Millisecond,
      DialTimeout:         time.Duration(PullTimeout)*time.Millisecond,
      RequestTimeout:      time.Duration(HoldTimeout)*time.Millisecond,
      Params:              query,
      Context:             cs.ctx,
   }
   res,err:=grequests.Get(config.Config.Apollo.Host+":"+config.Config.Apollo.Port+"/notifications/v2",options)
   defer func() {
		if res != nil {
			_ = res.Close()
		}
	}()
   if err != nil {
   	  logrus.Error("fetch notifications error,reason is "+err.Error())
   	  return
   }

   if res == nil {
   	  logrus.Error("fetch notifications res is nil")
   	  return
   }

   if res.StatusCode!=200 &&res.StatusCode!=304 {
   	  logrus.Error("fetch notifications http code is "+string(res.StatusCode))
   	  return
   }

   if res.StatusCode == 304 {
   	  return
   }

    updateNamespaceInfo:= make([]map[string]interface{},len(notificationSlice))
	content := res.Bytes()
	err = json.Unmarshal(content, &updateNamespaceInfo)
	if err != nil {
		logrus.Error("json decode failed")
		return
	}

	//update notifications
	cs.updateNotifications(updateNamespaceInfo)
	//fetch update configs
	cs.PullBatch()
}

func (cs *Apollo) updateNotifications(updateNamespaceInfo []map[string]interface{}){
	cs.mu.Lock()
	defer cs.mu.Unlock()
	notificationsMap:=cs.notifications
	for _,namespaceInfo := range updateNamespaceInfo {
		var notificationIdOld float64
		var namespaceOld string
		if notificationIdNew, ok := namespaceInfo["notificationId"]; ok {
			if reflect.TypeOf(notificationIdNew).Kind() == reflect.Float64 {
				notificationIdOld = notificationIdNew.(float64)
			} else {
				notificationIdOld = -1
			}
		}
		if notificationIdOld == -1 {
			continue
		}

		if namespaceName, ok := namespaceInfo["namespaceName"]; ok {
			if reflect.TypeOf(namespaceName).Kind() == reflect.String {
				namespaceOld = namespaceName.(string)
			} else {
				namespaceOld = ""
			}
		}
		if namespaceOld == "" {
			continue
		}

		if _,ok:=notificationsMap[namespaceOld];!ok{
			notificationsMap[namespaceOld] = &Notification{}
			notificationsMap[namespaceOld].NamespaceName = namespaceOld
			notificationsMap[namespaceOld].NotificationId = notificationIdOld
		}else{
			notificationsMap[namespaceOld].NotificationId = notificationIdOld
		}
	}
}

func (cs *Apollo) getNotificationsSlice() []*Notification {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	namespaceInfo := len(cs.notifications)
	if namespaceInfo == 0 {
		return nil
	}

	notificationsSlice:= make([]*Notification, namespaceInfo)
	var i = 0
	for _, version := range cs.notifications {
		notificationsSlice[i] = version
		i++
	}

	return notificationsSlice
}


func (cs *Apollo) PullBatch(){
   cs.mu.RLock()
   defer cs.mu.RUnlock()
   for namespace,_:=range cs.notifications{
   	  go cs.pullWithCacheOrNot(namespace,false)
   }
}

func (cs *Apollo) pullWithCacheOrNot(namespace string,cache bool){
	query:=make(map[string]string)
	query["clientIp"] = cs.getLocalIp()
	var uri string
	if cache {
		uri = fmt.Sprintf("/configfiles/json/%s/%s/%s",
			config.Config.Apollo.AppId,
			config.Config.Apollo.ClusterName,
			namespace,
		)
	} else {
		uri = fmt.Sprintf("/configs/%s/%s/%s",
			config.Config.Apollo.AppId,
			config.Config.Apollo.ClusterName,
			namespace,
		)
	}
	releaseKey:=cs.getReleaseKey(namespace)
	if !cache{
		query["releaseKey"] = releaseKey
	}

	options:=&grequests.RequestOptions{
		TLSHandshakeTimeout: time.Duration(PullTimeout)*time.Millisecond,
		DialTimeout:         time.Duration(PullTimeout)*time.Millisecond,
		RequestTimeout:      time.Duration(PullTimeout)*time.Millisecond,
		Params:              query,
		Context:             cs.ctx,
	}

	res,err:=grequests.Get(config.Config.Apollo.Host+":"+config.Config.Apollo.Port+uri,options)

	defer func() {
		if res!=nil {
			_=res.Close()
		}
	}()

	if err != nil {
		logrus.Error("fetch configs failed ,error is  "+err.Error())
		return
	}

	if res == nil {
		logrus.Error("fetch configs failed ,body is empty")
		return
	}

	if res.StatusCode!=200&&res.StatusCode!=304 {
		logrus.Errorf("fetch configs failed ,status code is %s  ",res.StatusCode)
		return
	}

	if res.StatusCode==304 {
		logrus.Info("config has't been changed ,we dont need update")
		return
	}

	newConfigurations := make(map[string]interface{})
	response := res.Bytes()
	err = json.Unmarshal(response, &newConfigurations)
	if err != nil {
		logrus.Error("json decode failed ,error is "+err.Error())
		return
	} else {
		if cache {
			config := cs.getConfigurationsFromFile(namespace)
			if config != nil {
				config["configurations"] = newConfigurations
				newContent, _ := json.Marshal(config)

				cs.updateFile(namespace,string(newContent))

				logrus.Info("cache config success")
			}
		} else {
			if newReleaseKey, ok := newConfigurations["releaseKey"]; ok {
				if reflect.TypeOf(newReleaseKey).Kind() == reflect.String {
					logrus.Info("get config success, release key is "+newReleaseKey.(string))
				}
			} else {
				logrus.Error("config content is emptu")
				return
			}
			cs.updateFile(namespace, string(response))
		}
	}
}

func (cs *Apollo) updateFile(namespace string,content string){
	//write file
	fileName :=  fmt.Sprintf("%s/%s/apollo.cache.%s.json", config.Config.Apollo.ApolloDir, config.Config.Apollo.AppId, namespace)
	dir:=filepath.Dir(fileName)
	_,err:=os.Stat(dir)
	if err!=nil{
		if os.IsNotExist(err) {
			if err:=os.MkdirAll(dir,0777);err!=nil{
				logrus.Error("create config dir failed,error is "+err.Error())
				return
			}
		}
	}

	if err:=os.WriteFile(fileName,[]byte(content),0644);err!=nil{
		logrus.Error("write config to file failed ,error is "+err.Error())
		return
	}
}


func (cs *Apollo) getReleaseKey(namespace string) string{
	config := cs.getConfigurationsFromFile(namespace)
	if config == nil {
		return ""
	}

	if releaseKey, ok := config["releaseKey"]; ok {
		if reflect.TypeOf(releaseKey).Kind() == reflect.String {
			return releaseKey.(string)
		}
	}

	return ""
}

func (cs *Apollo) getConfigurationsFromFile(namespace string) map[string]interface{} {
	file :=  fmt.Sprintf("%s/%s/apollo.cache.%s.json", config.Config.Apollo.ApolloDir, config.Config.Apollo.AppId, namespace)
	_, err := os.Stat(file)

	if err!=nil{
		return nil
	}

	content, err := ioutil.ReadFile(file)
	if err != nil {
		logrus.Error("read file failed err s "+err.Error())
		return nil
	}

	configInfo := make(map[string]interface{})
	err = json.Unmarshal(content, &configInfo)
	if err != nil {
		logrus.Errorf("json decode failed ,err is "+err.Error())
	}

	return configInfo
}

func (cs *Apollo) getLocalIp() string{
	adders,err:=net.Interfaces()
	if err!=nil{
		logrus.Error("fetch local ip failed ,error is "+err.Error())
		return ""
	}
	var localIp string
	for _,v:=range adders {
		//for mac and linux
		if v.Name == "en0" || v.Name == "eth0" {
			addr,_:=v.Addrs()
			localIp = strings.Split(addr[0].String(),"/")[0]
			break
		}
	}
	return localIp
}
