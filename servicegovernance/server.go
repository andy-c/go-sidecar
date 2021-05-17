/**
  @description:go-sidecar
  @author: Angels lose their hair
  @date: 2021/5/12
  @version:v1
**/
package servicegovernance

import (
	"context"
	"github.com/caarlos0/env/v6"
	"go-sidecar/config"
	"go-sidecar/configcenter"
	"go-sidecar/log"
	"go-sidecar/registercenter"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var Singleton *Server

var once sync.Once
var WaitGroup sync.WaitGroup

type Server struct {
	Ctx context.Context
	Cancel context.CancelFunc
}

//Start Application
func (s *Server) Run(){
	//init config from env for config center
	if err:=env.Parse(&config.Config);err!=nil{
		panic("error : parse env failed,reason is "+err.Error())
	}
	//init logger
	log.InitLogger()
	//init global context and cancel func
	s.signal()
	//start config center
	configcenter.NewApollo(s.Ctx)
	//start register center
	registercenter.NewEureka(s.Ctx)
}

//handle signal
func (s *Server) signal(){
	//handler signal
	quit := make(chan os.Signal)
	signal.Notify(quit,syscall.SIGTERM,syscall.SIGQUIT,syscall.SIGINT,syscall.SIGHUP)
	go func(quit chan os.Signal){
		defer os.Exit(0)
		defer close(quit)
		for{
			select {
			   case <-quit:
			   	    s.close()
			   	    return
			}
		}
	}(quit)
}

//close context
func (s *Server) close(){
	s.Cancel()
	config.WaitGroup.Wait()
}

func New() *Server{
	ctx ,cancel:= context.WithCancel(context.Background())
	once.Do(func() {
		Singleton=&Server{
			Ctx: ctx,
			Cancel: cancel,
		}
	})
	return Singleton
}

func getInstance() *Server{
	return Singleton
}