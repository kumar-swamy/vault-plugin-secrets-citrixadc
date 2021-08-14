package util

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/kumar-swamy/vault-plugin-secrets-citrixadc/plugin/client"
)

func NewSecretsClient(logger hclog.Logger) *SecretsClient {
	return &SecretsClient{adcClient: client.NewClient(logger)}
}

// SecretsClient wraps a *activeDirectory.activeDirectoryClient to expose just the common convenience methods needed by the ad secrets backend.
type SecretsClient struct {
	adcClient *client.Client
}

func (c *SecretsClient) Get(conf *client.ADCConf, userName string) (*client.UserEntry, error) {

	entry, err := c.adcClient.Get(conf, userName)
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, fmt.Errorf("unable to find Username named %s in ADC ", userName)
	}

	return entry, nil
}

func (c *SecretsClient) UpdatePassword(conf *client.ADCConf, userName string, newPassword string) error {

	return c.adcClient.UpdatePassword(conf, userName, newPassword)
}

func (c *SecretsClient) UpdateRootPassword(conf *client.ADCConf, adminUser string, newPassword string) error {

	return c.adcClient.UpdatePassword(conf, adminUser, newPassword)
}
