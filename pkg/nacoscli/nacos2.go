package nacoscli

import (
	"bytes"
	"fmt"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/spf13/viper"
)

func NewClient(url, namespace string, port uint64) (config_client.IConfigClient, error) {
	sc := []constant.ServerConfig{
		*constant.NewServerConfig(url, port, constant.WithContextPath("/nacos")),
	}
	cc := *constant.NewClientConfig(
		constant.WithNamespaceId(namespace),
		constant.WithNotLoadCacheAtStart(true),
	)
	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		},
	)
	return client, err
}

func GetAndWatchConfig(client config_client.IConfigClient, dataId, group, configType string, config interface{}) error {
	content, err := client.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
	if err != nil {
		return err
	}
	fmt.Println("config content", content)
	viper.SetConfigType(configType)
	if err = viper.ReadConfig(bytes.NewBufferString(content)); err != nil {
		return err
	}
	if err = viper.Unmarshal(config); err != nil {
		return err
	}
	err = client.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
		OnChange: func(namespace, group, dataId, data string) {
			fmt.Println("config change", data)
			viper.ReadConfig(bytes.NewBufferString(data))
			viper.Unmarshal(config)
		},
	})
	if err != nil {
		return err
	}
	return nil
}
