package plugin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/kumar-swamy/vault-plugin-secrets-citrixadc/plugin/client"
)

const (
	configPath            = "config"
	configStorageKey      = "config"
	defaultPasswordLength = 16
)

func readConfig(ctx context.Context, storage logical.Storage) (*configuration, error) {
	entry, err := storage.Get(ctx, configStorageKey)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}
	config := &configuration{}
	if err := entry.DecodeJSON(config); err != nil {
		return nil, err
	}
	return config, nil
}

func writeConfig(ctx context.Context, storage logical.Storage, config *configuration) (err error) {
	entry, err := logical.StorageEntryJSON(configStorageKey, config)
	if err != nil {
		return fmt.Errorf("unable to marshal config to JSON: %w", err)
	}
	if err := storage.Put(ctx, entry); err != nil {
		return fmt.Errorf("unable to store config: %w", err)
	}
	return nil
}

func (b *backend) pathConfig() *framework.Path {
	return &framework.Path{
		Pattern: configPath,
		Fields:  b.configFields(),
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: b.configUpdateOperation,
			logical.ReadOperation:   b.configReadOperation,
			logical.DeleteOperation: b.configDeleteOperation,
		},
		HelpSynopsis:    configHelpSynopsis,
		HelpDescription: configHelpDescription,
	}
}

func (b *backend) configFields() map[string]*framework.FieldSchema {
	fields := client.ConfigFields()
	fields["ttl"] = &framework.FieldSchema{
		Type:        framework.TypeDurationSecond,
		Description: "In seconds, the default password time-to-live.",
	}
	fields["max_ttl"] = &framework.FieldSchema{
		Type:        framework.TypeDurationSecond,
		Description: "In seconds, the maximum password time-to-live.",
	}
	fields["password_policy"] = &framework.FieldSchema{
		Type:        framework.TypeString,
		Description: "Name of the password policy to use to generate passwords.",
	}

	// Deprecated fields
	fields["length"] = &framework.FieldSchema{
		Type:        framework.TypeInt,
		Default:     defaultPasswordLength,
		Description: "The desired length of passwords that Vault generates. Mutually exclusive with password_policy field",
		Deprecated:  false,
	}
	fields["formatter"] = &framework.FieldSchema{
		Type:        framework.TypeString,
		Description: `Text to insert the password into, ex. "customPrefix{{PASSWORD}}customSuffix" Mutually exclusive with password_policy field`,
		Deprecated:  false,
	}
	return fields
}

func (b *backend) configUpdateOperation(ctx context.Context, req *logical.Request, fieldData *framework.FieldData) (*logical.Response, error) {
	// Build and validate the ldap conf.
	citrixAdcConf, err := client.NewConfigEntry(nil, fieldData)
	if err != nil {
		return nil, err
	}
	if err := citrixAdcConf.Validate(); err != nil {
		return nil, err
	}

	// Build the password conf.
	ttl := fieldData.Get("ttl").(int)
	maxTTL := fieldData.Get("max_ttl").(int)

	length := fieldData.Get("length").(int)
	formatter := fieldData.Get("formatter").(string)
	passwordPolicy := fieldData.Get("password_policy").(string)

	if ttl == 0 {
		ttl = int(b.System().DefaultLeaseTTL().Seconds())
	}
	if maxTTL == 0 {
		maxTTL = int(b.System().MaxLeaseTTL().Seconds())
	}
	if ttl > maxTTL {
		return nil, errors.New("ttl must be smaller than or equal to max_ttl")
	}
	if ttl < 1 {
		return nil, errors.New("ttl must be positive")
	}
	if maxTTL < 1 {
		return nil, errors.New("max_ttl must be positive")
	}

	passwordConf := passwordConf{
		TTL:            ttl,
		MaxTTL:         maxTTL,
		Length:         length,
		Formatter:      formatter,
		PasswordPolicy: passwordPolicy,
	}
	err = passwordConf.validate()
	if err != nil {
		return nil, err
	}

	config := configuration{
		PasswordConf: passwordConf,
		ADCConf: &client.ADCConf{
			ConfigEntry: citrixAdcConf,
		},
	}
	err = writeConfig(ctx, req.Storage, &config)
	if err != nil {
		return nil, err
	}

	// Respond with a 204.
	return nil, nil
}

func (b *backend) configReadOperation(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	config, err := readConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}

	// NOTE:
	// "password" is intentionally not returned by this endpoint,
	// as we lean away from returning sensitive information unless it's absolutely necessary.
	// Also, we don't return the full ADConf here because not all parameters are used by this engine.
	configMap := map[string]interface{}{
		"url":          config.ADCConf.Url,
		"insecure_tls": config.ADCConf.InsecureTLS,
		"certificate":  config.ADCConf.Certificate,
	}
	if !config.ADCConf.LastBindPasswordRotation.Equal(time.Time{}) {
		configMap["last_bind_password_rotation"] = config.ADCConf.LastBindPasswordRotation
	}

	for k, v := range config.PasswordConf.Map() {
		configMap[k] = v
	}

	resp := &logical.Response{
		Data: configMap,
	}
	return resp, nil
}

func (b *backend) configDeleteOperation(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	if err := req.Storage.Delete(ctx, configStorageKey); err != nil {
		return nil, err
	}
	return nil, nil
}

const (
	configHelpSynopsis = `
Configure the Citrix ADC to connect to, along with password options.
`
	configHelpDescription = `
This endpoint allows you to configure the Citrix ADC to connect to and its
configuration options. When you add, update, or delete a config, it takes
immediate effect on all subsequent actions. It does not apply itself to roles
or creds added in the past.

The AD URL can use either the "http://" or "https://" schema. In the former
case, an unencrypted connection will be made with a default port of 443
`
)
