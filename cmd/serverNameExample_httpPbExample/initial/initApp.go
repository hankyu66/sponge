// Package initial is the package that starts the service to initialize the service, including
// the initialization configuration, service configuration, connecting to the database, and
// resource release needed when shutting down the service.
package initial

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/zhufuyi/sponge/configs"
	"github.com/zhufuyi/sponge/internal/config"

	//"github.com/zhufuyi/sponge/internal/model"

	"github.com/zhufuyi/sponge/pkg/logger"
	"github.com/zhufuyi/sponge/pkg/nacoscli"
	"github.com/zhufuyi/sponge/pkg/stat"
	"github.com/zhufuyi/sponge/pkg/tracer"
)

var (
	version            string
	configFile         string
	enableConfigCenter bool
)

// Config initial app configuration
func Config() {
	initConfig()
	cfg := config.Get()

	// initializing log
	_, err := logger.Init(
		logger.WithLevel(cfg.Logger.Level),
		logger.WithFormat(cfg.Logger.Format),
		logger.WithSave(cfg.Logger.IsSave),
	)
	if err != nil {
		panic(err)
	}

	// initializing database
	//model.InitMysql()
	//model.InitCache(cfg.App.CacheType)

	// initializing tracing
	if cfg.App.EnableTrace {
		tracer.InitWithConfig(
			cfg.App.Name,
			cfg.App.Env,
			cfg.App.Version,
			cfg.Jaeger.AgentHost,
			strconv.Itoa(cfg.Jaeger.AgentPort),
			cfg.App.TracingSamplingRate,
		)
	}

	// initializing the print system and process resources
	if cfg.App.EnableStat {
		stat.Init(
			stat.WithLog(logger.Get()),
			stat.WithAlarm(), // invalid if it is windows, the default threshold for cpu and memory is 0.8, you can modify them
		)
	}
}

func initConfig() {
	flag.StringVar(&version, "version", "", "service Version Number")
	flag.BoolVar(&enableConfigCenter, "enable-cc", false, "whether to get from the configuration center, "+
		"if true, the '-c' parameter indicates the configuration center")
	flag.StringVar(&configFile, "c", "", "configuration file")
	flag.Parse()

	if enableConfigCenter {
		// get the configuration from the configuration center (first get the nacos configuration,
		// then read the service configuration according to the nacos configuration center)
		if configFile == "" {
			configFile = configs.Path("serverNameExample_cc.yml")
		}
		appConfig := &config.Config{}
		if err := initNacos(appConfig); err != nil {
			panic(fmt.Sprintf("Connect nacos failed, %s", err))
		}
		config.Set(appConfig)
	} else {
		// get configuration from local configuration file
		if configFile == "" {
			configFile = configs.Path("serverNameExample.yml")
		}
		err := config.Init(configFile)
		if err != nil {
			panic("init config error: " + err.Error())
		}
	}

	if version != "" {
		config.Get().App.Version = version
	}
	//fmt.Println(config.Show())
}

func initNacos(config interface{}) error {
	url := os.Getenv("NACOS_Url")
	namespace := os.Getenv("NACOS_Namespace")
	group := os.Getenv("NACOS_GroupName")
	dataId := os.Getenv("NACOS_DataId")
	// auth := os.Getenv("NACOS_Auth")
	// user := os.Getenv("NACOS_User")
	// password := os.Getenv("NACOS_Password")
	configType := os.Getenv("NACOS_ConfigType")
	port := func() uint64 {
		p, err := strconv.Atoi(os.Getenv("NACOS_Port"))
		if err != nil {
			return 80
		}
		return uint64(p)
	}()
	client, err := nacoscli.NewClient(url, namespace, port)
	if err != nil {
		fmt.Println("NewClient err", err)
		return err
	}
	if err := nacoscli.GetAndWatchConfig(client, dataId, group, configType, config); err != nil {
		fmt.Println("GetAndWatchConfig err", err)
		return err
	}
	return nil
}
