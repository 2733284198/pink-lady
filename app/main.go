// gin-skeleton: Typically gin-based web application's organizational structure
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"

	"github.com/axiaoxin/gin-skeleton/app/apis"
	"github.com/axiaoxin/gin-skeleton/app/common"
	"github.com/axiaoxin/gin-skeleton/app/middleware"
	"github.com/axiaoxin/gin-skeleton/app/models"
	"github.com/axiaoxin/gin-skeleton/app/utils"
	"github.com/fvbock/endless"
	raven "github.com/getsentry/raven-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sentry"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func init() {
	utils.InitViper([]utils.Option{
		utils.Option{"server.mode", "debug", "server mode: debug|test|release"},
		utils.Option{"server.bind", ":9090", "server bind address"},
		utils.Option{"log.level", "info", "log level: debug|info|warning|error|fatal|panic"},
		utils.Option{"log.formatter", "text", "log formatter: text|json"},
		utils.Option{"database.engine", "sqlite3", "database engine: mysql|postgres|sqlite3|mssql"},
		utils.Option{"database.address", "", "database address: host:port"},
		utils.Option{"database.name", "/tmp/gin-skeleton.db", "database name"},
		utils.Option{"database.username", "", "database username"},
		utils.Option{"database.password", "", "database password"},
		utils.Option{"database.max_idle_conns", 2, "sets the maximum number of connections in the idle connection pool."},
		utils.Option{"database.max_open_conns", 0, "sets the maximum number of open connections to the database."},
		utils.Option{"database.conn_max_life_minutes", 0, "sets the maximum amount of time(minutes) a connection may be reused."},
		utils.Option{"database.log_mode", true, "show detailed sql log"},
		utils.Option{"redis.mode", "single-instance", "redis mode: single-instance|sentinel|cluster"},
		utils.Option{"redis.address", "localhost:6379", "redis address, multiple sentinel/cluster addresses are separated by commas"},
		utils.Option{"redis.password", "", "redis password"},
		utils.Option{"redis.db", 0, "redis default db"},
		utils.Option{"redis.master", "", "redis sentinel master name"},
		utils.Option{"sentry.dsn", "", "sentry dsn"},
		utils.Option{"sentry.onlycrashes", "", "sentry only send crash reporting"},
	})

	utils.InitLogrus(viper.GetString("log.level"), viper.GetString("log.formatter"))
	utils.InitGormDB(viper.GetString("database.engine"), viper.GetString("database.address"), viper.GetString("database.name"), viper.GetString("database.username"), viper.GetString("database.password"), viper.GetInt("database.max_idle_conns"), viper.GetInt("database.max_open_conns"), viper.GetInt("database.conn_max_life_minutes"), viper.GetBool("log_mode"))
	models.Migrate()
	utils.InitRedis(viper.GetString("redis.mode"), viper.GetString("redis.address"), viper.GetString("redis.password"), viper.GetInt("redis.db"), viper.GetString("redis.master"))
}

func main() {
	defer utils.DB.Close()
	// TODO: imp in cli
	version := pflag.Bool("version", false, "show version")
	check := pflag.Bool("check", false, "check everything need to be checked")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
	if *version {
		fmt.Println(common.VERSION)
		os.Exit(0)
	}
	if *check {
		fmt.Println("I'm fine :)")
		os.Exit(0)
	}

	mode := strings.ToLower(viper.GetString("server.mode"))
	if mode == "debug" {
		gin.SetMode(gin.DebugMode)
	} else if mode == "test" {
		gin.SetMode(gin.TestMode)
	} else {
		gin.DisableConsoleColor()
		gin.SetMode(gin.ReleaseMode)
	}

	app := gin.New()
	app.Use(middleware.ErrorHandler())
	app.Use(cors.Default())
	app.Use(middleware.RequestID())
	app.Use(middleware.GinLogrus())
	sentryDSN := viper.GetString("sentry.dsn")
	if sentryDSN != "" {
		raven.SetDSN(sentryDSN)
		app.Use(sentry.Recovery(raven.DefaultClient, viper.GetBool("sentry.onlycrashes")))
	}

	apis.RegisterRoutes(app)

	bind := viper.GetString("server.bind")
	if runtime.GOOS == "windows" {
		app.Run(bind)
	} else {
		server := endless.NewServer(bind, app)
		server.BeforeBegin = func(addr string) {
			logrus.Infof("Gin server is listening and serving HTTP on %s (pids: %d)", addr, syscall.Getpid())
		}
		server.ListenAndServe()
	}
}
