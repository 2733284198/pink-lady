package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/axiaoxin-com/pink-lady/apis"

	"github.com/axiaoxin-com/goutils"
	"github.com/axiaoxin-com/logging"
	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	// DefaultServerAddr 默认运行地址
	DefaultServerAddr = ":4869"
	// DefaultServerReadTimeout 服务器从 accept 到读取 body 的超时时间（秒）
	DefaultServerReadTimeout = 5
	// DefaultServerWriteTimeout 服务器从 accept 到写 response 的超时时间（秒）
	DefaultServerWriteTimeout = 5
	// DefaultBasicAuthUsername BasicAuth 默认用户名
	DefaultBasicAuthUsername = "admin"
	// DefaultBasicAuthPassword BasicAuth 默认密码
	DefaultBasicAuthPassword = "admin"
)

func init() {
	// 加载配置文件到 viper
	workdir, err := os.Getwd()
	if err != nil {
		logging.Fatal(nil, "get workdir failed", zap.Error(err))
	}
	configPath := flag.String("p", workdir, "path of config file")
	configName := flag.String("c", "config", "name of config file (no suffix)")
	configType := flag.String("t", "toml", "type of config file (the file format suffix)")
	flag.Parse()
	if err := goutils.InitViper(*configPath, *configName, *configType, func(e fsnotify.Event) {
		logging.Warn(nil, "Config file changed:"+e.Name)
	}); err != nil {
		panic("Init viper error:" + err.Error())
	}

	// 设置配置默认值
	viper.SetDefault("server.mode", gin.ReleaseMode)
	viper.SetDefault("server.pprof", true)
	viper.SetDefault("server.addr", DefaultServerAddr)
	viper.SetDefault("server.read_timeout", DefaultServerReadTimeout)
	viper.SetDefault("server.write_timeout", DefaultServerWriteTimeout)
	viper.SetDefault("basic_auth.username", DefaultBasicAuthUsername)
	viper.SetDefault("basic_auth.password", DefaultBasicAuthPassword)
}

// 根据配置创建并运行使用 gin 处理请求的 http server
func main() {
	// 创建 gin engine
	middlewares := []gin.HandlerFunc{}
	ginEngine := goutils.NewGinEngine(viper.GetString("server.mode"), viper.GetBool("server.pprof"), middlewares...)

	// 注册 apis 到 gin engine
	apis.Register(ginEngine)

	addr := viper.GetString("server.addr")
	readTimeout := viper.GetInt("server.handler_timeout")
	writeTimeout := viper.GetInt("server.handler_timeout")
	srv := &http.Server{
		Addr:         addr,
		Handler:      ginEngine,
		ReadTimeout:  time.Duration(readTimeout) * time.Second,
		WriteTimeout: time.Duration(writeTimeout) * time.Second,
	}

	// 关闭 db 连接
	defer goutils.CloseGormInstances()

	// 启动 http server
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Fatal(nil, "Server start error:"+err.Error())
		}
	}()
	logging.Info(nil, "Server is listening and serving on "+srv.Addr)

	// 监听中断信号，WriteTimeout时间后优雅关闭服务
	// syscall.SIGTERM 不带参数的 kill 命令
	// syscall.SIGINT ctrl-c kill -2
	// syscall.SIGKILL 是 kill -9 无法捕获这个信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logging.Infof(nil, "Server will shutdown after %d seconds", writeTimeout)

	// 创建一个 context 用于通知 server 有 writeTimeout 秒的时间结束当前正在处理的请求
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(writeTimeout)*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logging.Fatal(nil, "Server shutdown with error: "+err.Error())
	}
	logging.Info(nil, "Server shutdown")
}