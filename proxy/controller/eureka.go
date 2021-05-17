/**
  @description:go-sidecar
  @author: Angels lose their hair
  @date: 2021/5/17
  @version:v1
**/
package controller

import (
	"github.com/gin-gonic/gin"
	"go-sidecar/registercenter"
	"net/http"
)

func EurekaInfo(c *gin.Context){
     c.JSON(200,registercenter.Singleton.GetInstanceInfo())
}

func EurekaStatus(c *gin.Context){
	c.JSON(200, registercenter.Singleton.Status)
}

func EurekaInstances(c *gin.Context){
	c.JSON(http.StatusOK,registercenter.Singleton.GetInstances())
}