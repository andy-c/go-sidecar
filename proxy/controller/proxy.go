/**
  @description:go-sidecar
  @author: Angels lose their hair
  @date: 2021/5/17
  @version:v1
**/
package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/levigross/grequests"
	"github.com/sirupsen/logrus"
	"go-sidecar/config"
	"go-sidecar/registercenter"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	PullTimeout = 4000
)

type ServiceInfo struct{
	ServiceName string `form:"serviceName"`
	Method string `form:"method"`
	MethodName string `form:"methodName"`
}

func Proxy(c *gin.Context){
	if config.Config.IsConsulOrEureka == "consul" {
		ProxyConsul(c)
	}else if config.Config.IsConsulOrEureka == "eureka"{
		ProxyEureka(c)
	}
}

func ProxyEureka(c *gin.Context){
   var response *grequests.Response
   var err error
   var scheme string
   var port int
   var ip string

   defer func() {
   	  if response != nil {
   	  	 response.Close()
	  }
   }()
   sInfo:=&ServiceInfo{}
   if c.ShouldBindQuery(sInfo) != nil {
   	  logrus.Errorf("params incorrectly,method is %s,serviceName is %s",sInfo.Method,sInfo.ServiceName)
   	  c.JSON(http.StatusBadRequest,gin.H{
   	  	"msg":"bad request params incorrectly",
   	  	"data":"",
	  })
   	  return
   }
   //random a service
   serviceInstance:=registercenter.Singleton.Random(sInfo.ServiceName)

   if serviceInstance == nil {
   	   c.JSON(http.StatusNotFound,gin.H{
   	   	  "msg":"service not found",
   	   	  "data":"",
	   })
   	   return
   }

   if serviceInstance.Status != "UP" {
   	  c.JSON(http.StatusServiceUnavailable,gin.H{
   	  	"msg":"service unavailable",
   	  	"data":"",
	  })
	   return
   }

   if serviceInstance.SecurePort.Enabled == "true"{
   	   port=serviceInstance.SecurePort.Port
   	   scheme="https://"
   }else{
   	   port=serviceInstance.Port.Port
   	   scheme="http://"
   }

   if serviceInstance.IpAddr!=""{
   	  ip = serviceInstance.IpAddr
   }else{
   	  ip = serviceInstance.HostName
   }

   if ip == "" {
   	c.JSON(http.StatusServiceUnavailable,gin.H{
   		"msg":"service unavailable",
   		"data":"",
	})
   	return
   }
   //all pass
   //so we can do the request
   proxyHeader:=c.Request.Header
   headers:=make(map[string]string)
   for k,v:=range proxyHeader{
   	  headers[k] = strings.Join(v,",")
   }

   options:=&grequests.RequestOptions{
	   TLSHandshakeTimeout: time.Duration(PullTimeout)*time.Millisecond,
	   DialTimeout:         time.Duration(PullTimeout)*time.Millisecond,
	   RequestTimeout:      time.Duration(PullTimeout)*time.Millisecond,
	   Headers: headers,
	   Context:	c.Request.Context(),
	   RequestBody: c.Request.Body,
   }
   response,err= grequests.Req(strings.ToUpper(sInfo.Method),scheme+ip+":"+strconv.Itoa(port)+"/"+sInfo.MethodName,options)
   if err!=nil{
   	  logrus.Error("call remote api failed,error is "+err.Error())
   	  c.JSON(http.StatusInternalServerError,gin.H{
   	  	 "msg":err.Error(),
   	  	 "upstreamCode":response.StatusCode,
   	  	 "data":"",
	  })
   	  return
   }

   if response == nil ||response.StatusCode!=200{
   	 logrus.Error("call remote api response is empty or code is "+strconv.Itoa(response.StatusCode))
   	 c.JSON(http.StatusInternalServerError,gin.H{
   	 	"msg":"response is empty",
   	 	"upstreamCode":response.StatusCode,
   	 	"data":"",
	 })
   	 return
   }else{
   	   c.JSON(http.StatusOK,gin.H{
   	   	  "msg":"",
   	   	  "upstreamCode":response.StatusCode,
   	   	  "data":response.String(),
	   })
	   return
   }

}

func ProxyConsul(c *gin.Context){
	var response *grequests.Response
	var err error
	var scheme string
	var port int
	var ip string

	defer func() {
		if response != nil {
			response.Close()
		}
	}()
	sInfo:=&ServiceInfo{}
	if c.ShouldBindQuery(sInfo) != nil {
		logrus.Errorf("params incorrectly,method is %s,serviceName is %s",sInfo.Method,sInfo.ServiceName)
		c.JSON(http.StatusBadRequest,gin.H{
			"msg":"bad request params incorrectly",
			"data":"",
		})
		return
	}
	//random a service
	serviceInstance:=registercenter.ConsulSingleton.Random(sInfo.ServiceName)
	if serviceInstance == nil {
		c.JSON(http.StatusNotFound,gin.H{
			"msg":"service not found",
			"data":"",
		})
		return
	}

	ip = serviceInstance.Address
	port = serviceInstance.Port
	scheme = "http://"

	//all pass
	//so we can do the request
	proxyHeader:=c.Request.Header
	headers:=make(map[string]string)
	for k,v:=range proxyHeader{
		headers[k] = strings.Join(v,",")
	}

	options:=&grequests.RequestOptions{
		TLSHandshakeTimeout: time.Duration(PullTimeout)*time.Millisecond,
		DialTimeout:         time.Duration(PullTimeout)*time.Millisecond,
		RequestTimeout:      time.Duration(PullTimeout)*time.Millisecond,
		Headers: headers,
		Context:	c.Request.Context(),
		RequestBody: c.Request.Body,
	}
	response,err= grequests.Req(strings.ToUpper(sInfo.Method),scheme+ip+":"+strconv.Itoa(port)+"/"+sInfo.MethodName,options)
	if err!=nil{
		logrus.Error("call remote api failed,error is "+err.Error())
		c.JSON(http.StatusInternalServerError,gin.H{
			"msg":err.Error(),
			"upstreamCode":response.StatusCode,
			"data":"",
		})
		return
	}

	if response == nil ||response.StatusCode!=200{
		logrus.Error("call remote api response is empty or code is "+strconv.Itoa(response.StatusCode))
		c.JSON(http.StatusInternalServerError,gin.H{
			"msg":"response is empty",
			"upstreamCode":response.StatusCode,
			"data":"",
		})
		return
	}else{
		c.JSON(http.StatusOK,gin.H{
			"msg":"",
			"upstreamCode":response.StatusCode,
			"data":response.String(),
		})
		return
	}
}

func Health(c *gin.Context){
	c.String(http.StatusOK,"ok")
}
