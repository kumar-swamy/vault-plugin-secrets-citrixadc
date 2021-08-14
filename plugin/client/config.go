package client

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/sdk/framework"
)

// ConfigFields returns all the config fields that can potentially be used by the Citrix ADC client.
// Not all fields will be used by every integration.
func ConfigFields() map[string]*framework.FieldSchema {
	return map[string]*framework.FieldSchema{
		"url": {
			Type:        framework.TypeString,
			Description: "Citrix ADC URL to connect to. Multiple URLs can be specified by concatenating them with commas; they will be tried in-order.",
			DisplayAttrs: &framework.DisplayAttributes{
				Name: "URL",
			},
		},

		"admin_username": {
			Type:        framework.TypeString,
			Description: "A user of citrix ADC which is used to set the password for another account. Make sure to create a seperate user with least privilege",
			DisplayAttrs: &framework.DisplayAttributes{
				Name: "Admin Username",
			},
		},

		"admin_password": {
			Type:        framework.TypeString,
			Description: "password for admin_username of citrix ADC which is used to set the password for another account. Make sure to create a seperate user with least privilege",
			DisplayAttrs: &framework.DisplayAttributes{
				Name: "Admin Password",
			},
		},

		"certificate": {
			Type:        framework.TypeString,
			Description: "CA certificate to use when verifying Citrix ADC server certificate, must be x509 PEM encoded (optional)",
			DisplayAttrs: &framework.DisplayAttributes{
				Name:     "CA certificate",
				EditType: "file",
			},
		},

		"insecure_tls": {
			Type:        framework.TypeBool,
			Description: "Skip Citrix ADC SSL Certificate verification - VERY insecure (optional)",
			DisplayAttrs: &framework.DisplayAttributes{
				Name: "Insecure TLS",
			},
		},

		"request_timeout": {
			Type:        framework.TypeDurationSecond,
			Description: "Timeout, in seconds, for the connection when making requests against the server before returning back an error.",
			Default:     "90s",
		},
	}
}

/*
 * Creates and initializes a ConfigEntry object with its default values,
 * as specified by the passed schema.
 */
func NewConfigEntry(existing *ConfigEntry, d *framework.FieldData) (*ConfigEntry, error) {
	var hadExisting bool
	var cfg *ConfigEntry

	if existing != nil {
		cfg = existing
		hadExisting = true
	} else {
		cfg = new(ConfigEntry)
	}

	if _, ok := d.Raw["url"]; ok || !hadExisting {
		cfg.Url = strings.ToLower(d.Get("url").(string))
	}

	if _, ok := d.Raw["certificate"]; ok || !hadExisting {
		certificate := d.Get("certificate").(string)
		if certificate != "" {
			if err := validateCertificate([]byte(certificate)); err != nil {
				return nil, errwrap.Wrapf("failed to parse server tls cert: {{err}}", err)
			}
		}
		cfg.Certificate = certificate
	}

	if _, ok := d.Raw["insecure_tls"]; ok || !hadExisting {
		cfg.InsecureTLS = d.Get("insecure_tls").(bool)
	}

	if _, ok := d.Raw["request_timeout"]; ok || !hadExisting {
		cfg.RequestTimeout = d.Get("request_timeout").(int)
	}

	if _, ok := d.Raw["admin_username"]; ok || !hadExisting {
		cfg.AdminUserName = d.Get("admin_username").(string)
	}

	if _, ok := d.Raw["admin_password"]; ok || !hadExisting {
		cfg.AdminPassword = d.Get("admin_password").(string)
	}

	return cfg, nil
}

type ConfigEntry struct {
	Url            string `json:"url"`
	Certificate    string `json:"certificate"`
	InsecureTLS    bool   `json:"insecure_tls"`
	RequestTimeout int    `json:"request_timeout"`
	AdminUserName  string `json:"admin_username"`
	AdminPassword  string `json:"admin_password"`
}

func (c *ConfigEntry) Map() map[string]interface{} {
	m := c.PasswordlessMap()
	m["admin_password"] = c.AdminPassword
	return c.PasswordlessMap()
}

func (c *ConfigEntry) PasswordlessMap() map[string]interface{} {
	m := map[string]interface{}{
		"url":             c.Url,
		"certificate":     c.Certificate,
		"insecure_tls":    c.InsecureTLS,
		"request_timeout": c.RequestTimeout,
		"admin_username":  c.AdminUserName,
	}
	return m
}

func validateCertificate(pemBlock []byte) error {
	block, _ := pem.Decode([]byte(pemBlock))
	if block == nil || block.Type != "CERTIFICATE" {
		return errors.New("failed to decode PEM block in the certificate")
	}
	_, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate %s", err.Error())
	}
	return nil
}

func (c *ConfigEntry) Validate() error {
	if len(c.Url) == 0 {
		return errors.New("at least one url must be provided")
	}

	if c.Certificate != "" {
		if err := validateCertificate([]byte(c.Certificate)); err != nil {
			return errwrap.Wrapf("failed to parse server tls cert: {{err}}", err)
		}
	}
	return nil
}

type ADCConf struct {
	*ConfigEntry
	LastBindPasswordRotation time.Time `json:"last_bind_password_rotation"`
}
