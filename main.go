/**
  @description:go-sidecar
  @author: Angels lose their hair
  @date: 2021/5/12
  @version:v1
**/
package main

import (
    proxy2 "go-sidecar/proxy"
    "go-sidecar/servicegovernance"
)

func main(){
    //service start
    serviceComponent:= servicegovernance.New()
    serviceComponent.Run()
    //proxy start
    proxy:=proxy2.NewProxy()
    proxy.Run()
}