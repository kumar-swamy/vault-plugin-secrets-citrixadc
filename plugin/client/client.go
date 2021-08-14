package client

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"

	"github.com/citrix/adc-nitro-go/resource/config/system"
	"github.com/citrix/adc-nitro-go/service"
	"github.com/hashicorp/go-hclog"
	"github.com/mitchellh/mapstructure"
)

func NewClient(logger hclog.Logger) *Client {
	return &Client{
		logger: &logger,
	}
}

type Client struct {
	logger *hclog.Logger
}

type UserEntry struct {
	userEntry *system.Systemuser
}

func (c *Client) NewNitroClientFromParams(params *ADCConf) (*service.NitroClient, error) {
	u, err := url.Parse(params.Url)
	if err != nil {
		return nil, fmt.Errorf("supplied URL %s is not a URL", params.Url)
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("supplied URL %s does not have a HTTP/HTTPS scheme", params.Url)
	}

	np := service.NitroParams{}
	np.Url = params.Url
	np.SslVerify = !params.InsecureTLS
	np.Timeout = params.RequestTimeout
	np.Username = params.AdminUserName
	np.Password = params.AdminPassword
	np.ServerName = host
	if params.Certificate != "" {
		file, err := ioutil.TempFile("tmp", "ca.cert")
		if err != nil {
			return nil, err
		}
		_, err = file.Write([]byte(params.Certificate))
		if err == nil {
			np.RootCAPath = file.Name()
		}
		defer os.Remove(file.Name())

	}
	nc, err := service.NewNitroClientFromParams(np)
	if err != nil {
		return nil, err
	}
	return nc, nil
}

func (c *Client) Get(cfg *ADCConf, userName string) (*UserEntry, error) {
	nc, err := c.NewNitroClientFromParams(cfg)
	if err != nil {
		return nil, err
	}

	obj, err := nc.FindResource("systemuser", userName)
	if err != nil {
		return nil, err
	}

	var entry UserEntry
	entry.userEntry = new(system.Systemuser)
	err = mapstructure.WeakDecode(&obj, entry.userEntry)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (c *Client) UpdatePassword(cfg *ADCConf, userName string, newPassword string) error {
	_, err := c.Get(cfg, userName)
	if err != nil {
		return err
	}

	nc, err := c.NewNitroClientFromParams(cfg)
	if err != nil {
		return err
	}

	_, err = nc.UpdateResource("systemuser", userName, system.Systemuser{Username: userName, Password: newPassword})

	if err != nil {
		return err
	}
	return nil
}
