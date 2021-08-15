# Vault Plugin: Citrix ADC vault plugin backend

This is a standalone backend plugin for use with [Hashicorp Vault](https://www.github.com/hashicorp/vault). This provides a functionality to rotate the Citrix ADC password automatically after a given TTL is elapsed by the vault which can be used by any clients to interact with citrix ADC. 

For example, [Citrix ingress controller](https://github.com/citrix/citrix-k8s-ingress-controller/) provides ingress controller functionality in the kubernetes which needs ADC credentials to configure the ingresses on ADC. With the help of this plugin, you can configure the vault to auto rotate the password after a given TTL is elapsed, thereby strengthening the security posture. 


## Quick Links
- Vault Website: https://www.vaultproject.io
- Citrix Ingress controller: https://github.com/citrix/citrix-k8s-ingress-controller/
- Main Project Github: https://www.github.com/hashicorp/vault

## Getting Started

This is a [Vault plugin](https://www.vaultproject.io/docs/internals/plugins.html)
and is meant to work with Vault. This guide assumes you have already installed Vault
and have a basic understanding of how Vault works.

Otherwise, first read this guide on how to [get started with Vault](https://www.vaultproject.io/intro/getting-started/install.html).

To learn specifically about how plugins work, see documentation on [Vault plugins](https://www.vaultproject.io/docs/internals/plugins.html).

## Developing

If you wish to work on this plugin, you'll first need
[Go](https://www.golang.org) installed on your machine
(version 1.10+ is *required*).

For local dev first make sure Go is properly installed, including
setting up a [GOPATH](https://golang.org/doc/code.html#GOPATH).
Next, clone this repository into
`$GOPATH/src/github.com/hashicorp/vault-plugin-secrets-citrixadc`.
You can then download any required build tools by bootstrapping your
environment:

```sh
$ make bootstrap
```

To compile a development version of this plugin, run `make` or `make dev`.
This will put the plugin binary in the `bin` and `$GOPATH/bin` folders. `dev`
mode will only generate the binary for your platform and is faster:

```sh
$ make
$ make dev
```

In order to build for different archtectures, you can run `make bin`:
```sh
$ make bin
```
Once the plugin is built, copy the relevant plugin module to the vault plugin directory.


## Usage



To use this plugin, you must load the plugin. Please refer the [documentation](https://www.vaultproject.io/docs/internals/plugins) on loading a vault plugin

Once the plugin is succesfully loaded, you can enable the plugin and start using it

```sh
vault secrets enable -path=citrixadc vault-plugin-secrets-citrixadc
Success! Enabled the vault-plugin-secrets-citrixadc secrets engine at: citrixadc/
```

Before configuring the end point, you must create an admin user in Citrix ADC which is used to set the password. This user must have the privilege to set the system user, so you must create a cmd policy and bind this to system user. 

```
# login to citrix ADC and run the following command
$ add system cmdPolicy edit-user ALLOW "(^\\S+\\s+system\\s+\\S+)|(^\\S+\\s+system\\s+\\S+\\s+.*)|(^\\S+\\s+system\\s+user)|(^\\S+\\s+system\\s+user\\s+.*)|(^(?!rm)\\S+\\s+system\\s+\\S+)|(^(?!rm)\\S+\\s+system\\s+\\S+\\s+.*)"

$ add system user vault-admin <password>
$ bind system user vault-admin edit-user 100 
```


Then you can access the `config` endpoint in the vault to configure the ADC details as shown below. 

```sh 
vault write citrixadc/config admin_username="vault-admin" admin_password=<password> insecure_tls=true url="https://x.x.x.x" max_ttl=24h ttl=1h
Success! Data written to: citrixadc/config
```

Now that you've configured the vault, you can setup the vault to dynamically rotate the password. 

Create a user in citrix ADC which is used by the client to interact with ADC.

```
# login to citrix ADC and create a user
$ add system user cic-user <password>
```
Now you can configure the `roles` endpoint to autorotate the password for this user. 

```
vault write citrixadc/roles/cic user_name=cic-user ttl=1h
```

Now you can read the current password using `creds` endpoint. This plugin will automatically rotate the password if the TTL is elpased. 
**NOTE** This will lazily rotate the password only when it is read and TTL is elapsed. If TTL is elpased but credential is not read, then password is not rotated

```
vault read citrixadc/creds/cic
Key                 Value
---                 -----
current_password    ?@09AZILjFmjcmtv
last_password       ?@09AZCUrltFhCic
username            cic-user
```

You can also rotate the password of admin user using `rotate-root` endpoint. This means only vault will have the password for admin user. 

```
vault write -force citrixadc/rotate-root
```

## Developing

If you wish to work on this plugin, you'll first need
[Go](https://www.golang.org) installed on your machine
(version 1.10+ is *required*).

For local dev first make sure Go is properly installed, including
setting up a [GOPATH](https://golang.org/doc/code.html#GOPATH).
Next, clone this repository into
`$GOPATH/src/github.com/hashicorp/vault-plugin-secrets-citrixadc`.
You can then download any required build tools by bootstrapping your
environment:

```sh
$ make bootstrap
```

To compile a development version of this plugin, run `make` or `make dev`.
This will put the plugin binary in the `bin` and `$GOPATH/bin` folders. `dev`
mode will only generate the binary for your platform and is faster:

```sh
$ make
$ make dev
```

Put the plugin binary into a location of your choice. This directory
will be specified as the [`plugin_directory`](https://www.vaultproject.io/docs/configuration/index.html#plugin_directory)
in the Vault config used to start the server.

```json
...
plugin_directory = "path/to/plugin/directory"
...
```

Start a Vault server with this config file:
```sh
$ vault server -config=path/to/config.json ...
...
```

Once the server is started, register the plugin in the Vault server's [plugin catalog](https://www.vaultproject.io/docs/internals/plugins.html#plugin-catalog):


#### Tests

If you are developing this plugin and want to verify it is still
functioning (and you haven't broken anything else), we recommend
running the tests.

To run the tests, run the following:

```sh
$ make test
```

You can also specify a `TESTARGS` variable to filter tests like so:

```sh
$ make test TESTARGS='--run=TestConfig'
```
