/**
  @description:go-sidecar
  @author: Angels lose their hair
  @date: 2021/5/17
  @version:v1
**/
package proxy

import (
	"fmt"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"go-sidecar/config"
	"go-sidecar/proxy/controller"
	"go-sidecar/proxy/middleware"
	"os"
)

type Proxy struct{
	Host string
	Port uint16
	Router *gin.Engine
}

func NewProxy() *Proxy{
	router:=gin.New()
	pprof.Register(router)
	proxy:=&Proxy{
		Host:config.Config.Server.Host,
		Port:config.Config.Server.Port,
		Router: router,
	}
	return proxy
}

func (p *Proxy) setRoutes(){
	p.Router.GET("/info",controller.EurekaInfo)
	p.Router.GET("/status",controller.EurekaStatus)
	p.Router.GET("/instances",controller.EurekaInstances)
    p.Router.GET("/proxy",controller.Proxy)
	p.Router.GET("/health",controller.Health)
}

func (p *Proxy) Run(){
	p.Router.Use(gin.Recovery(),middleware.GinLogger())
	p.setRoutes()
	err:=p.Router.Run(fmt.Sprintf("%s:%d",p.Host,p.Port))
	if err!=nil{
		fmt.Fprint(os.Stderr,"Proxy start failed")
		os.Exit(1)
	}

}