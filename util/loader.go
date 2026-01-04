package util

import (
	"encoding/base64"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func NewConfigLoader(loader *clientcmd.ClientConfigLoadingRules, configBase64Data string) *ConfigLoader {
	return &ConfigLoader{
		ClientConfigLoadingRules: loader,
		configBase64Data:         configBase64Data,
	}
}

type ConfigLoader struct {
	*clientcmd.ClientConfigLoadingRules
	configBase64Data string
}

func (cl ConfigLoader) Load() (*clientcmdapi.Config, error) {
	if cl.configBase64Data == "" {
		return cl.ClientConfigLoadingRules.Load()
	}
	data, err := base64.StdEncoding.DecodeString(cl.configBase64Data)
	if err != nil {
		return nil, err
	}
	cc, err := clientcmd.NewClientConfigFromBytes(data)
	if err != nil {
		return nil, err
	}
	cfg, err := cc.RawConfig()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
