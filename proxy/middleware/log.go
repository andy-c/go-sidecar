/**
  @description:go-sidecar
  @author: Angels lose their hair
  @date: 2021/5/17
  @version:v1
**/
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.Request.Header.Get("your-traceId")
		spanId := c.Request.Header.Get("your-spanId")
		upspan := c.Request.Header.Get("your-upSpan")

		if reqID == "" {
			reqID = genTraceId()
		}

		if spanId == "" {
			spanId = "0"
		}

		c.Set("your-traceId", reqID)

		serverIp := ""
		addresses, _ := net.InterfaceAddrs()
		for _, address := range addresses {
			if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil {
					serverIp = ipNet.IP.String()
				}
			}
		}

		fields := logrus.Fields{
			"traceId": reqID,
			"upSpan":   upspan,
			"spanId":  spanId,
			"rt":       0,
			"method":     c.Request.Method,
			"uri":      c.Request.URL.Path,
			"query":   c.Request.URL.RawQuery,
			"extend": map[string]interface{}{
				"user_agent":  c.Request.Header.Get("User-Agent"),
				"remote_addr": c.ClientIP(),
				"server_addr": serverIp,
			},
		}

		rs := time.Now().UnixNano() / 1e6
		c.Next()
		rt := time.Now().UnixNano()/1e6 - rs
		fields["rt"] = rt
		fields["code"] = c.Writer.Status()

		logrus.WithFields(fields).Info("request chain log")
	}
}

//sample uuid
//we can change the traceId func
func genTraceId() string{
	return  uuid.New().String()
}