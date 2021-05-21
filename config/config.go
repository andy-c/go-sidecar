/**
  @description:go-sidecar
  @author: Angels lose their hair
  @date: 2021/5/12
  @version:v1
**/
package config

import "sync"

var WaitGroup sync.WaitGroup
var Config=Configure{
	Apollo:&ApolloConfig{},
	Log:&LogConfig{},
	Eureka:&EurekaConfig{},
	Server:&ServerConfig{},
	Consul: &ConsulConfig{},
	IsConsulOrEureka:"consul",
}

type Configure struct {
	Apollo *ApolloConfig
	Log *LogConfig
	Eureka *EurekaConfig
	Server *ServerConfig
	Consul *ConsulConfig
	IsConsulOrEureka string `env:'REGISTER_CENTER_TYPE' envDefault:"eureka"`
}
type ApolloConfig struct {
	Host string `env:"APOLLO_HOST" envDefault:"http://apollo.com"`
	Port string `env:"APOLLO_PORT" envDefault:"8080"`
	ClusterName string `env:"APOLLO_CLUSTER_NAME" envDefault:"default"`
	Notifications string `env:"APOLLO_NOTIFICATIONS" envDefault:"eureka,log"`
	AppId string `env:"APOLLO_APP_ID" envDefault:"go-sidecar"`
	ApolloDir string `env:"APOLLO_DIR" envDefault:"/tmp/apollo"`
}
type LogConfig struct {
	Name string `env:"LOG_NAME" envDefault:"go-sidecar-v1"`
	Level string `env:"LOG_DEFAULT_LEVEL" envDefault:"info"`
	Format string `env:"LOG_FORMAT" envDefault:"json"`
	Path string  `env:"LOG_PATH" envDefault:"/tmp/log/"`
}

type EurekaConfig struct{
   AppName string `env:"APP_NAME" envDefault:"swoole-sidecar-v1"`
   HostName string `env:"HOST_NAME" envDefault:"127.0.0.1"`
   AppAddr string `env:"APP_ADDR" envDefault:"127.0.0.1"`
   AppPort string `env:"APP_PORT" envDefault:"8089"`
   AppSecurePort string `env:"APP_SECURE_PORT" envDefault:"443"`
   HomePageUri string `env:"HOME_PAGE_URI" envDefault:"info"`
   StatusPageUri string `env:"STATUS_PAGE_URI" envDefault:"status"`
   HealthCheckUri string `env:"HEALTH_CHECK_URI" envDefault:"health"`
   VipAddr string `env:"VIP_ADDR" envDefault:"127.0.0.1"`
   SecureVipAddr string `env:"SECURE_VIP_ADDR" envDefault:"127.0.0.1"`
   EurekaServerAddr string `env:"EUREKA_SERVER_ADDR" envDefault:"http://eureka.com"`
}

type ServerConfig struct {
   Host string `env:"SERVER_HOST" envDefault:"0.0.0.0"`
   Port uint16 `env:"SERVER_PORT" envDefault:"8089"`
}

type ConsulConfig struct{
	AppName string `env:"APP_NAME" envDefault:"GO-SIDECAR-V3"`
	HostName string `env:"HOST_NAME" envDefault:"127.0.0.1"`
	AppAddr string `env:"APP_ADDR" envDefault:"127.0.0.1"`
	AppPort string `env:"APP_PORT" envDefault:"8089"`
	HealthCheckUri string `env:"HEALTH_CHECK_URI" envDefault:"health"`
}
