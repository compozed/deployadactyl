![](https://raw.githubusercontent.com/compozed/images/master/deployadactyl_logo.png)

[![Release](https://img.shields.io/github/release/compozed/deployadactyl.svg)](https://github.com/compozed/deployadactyl/releases/latest)
[![CircleCI](https://circleci.com/gh/compozed/deployadactyl.svg?style=svg&circle-token=0eab8bce42440217fb24ffd8ffdc2b44932125d5)](https://circleci.com/gh/compozed/deployadactyl)
[![Go Report Card](https://goreportcard.com/badge/github.com/compozed/deployadactyl)](https://goreportcard.com/report/github.com/compozed/deployadactyl)
[![codecov](https://codecov.io/gh/compozed/deployadactyl/branch/master/graph/badge.svg?token=r9yd1cwtbH)](https://codecov.io/gh/compozed/deployadactyl)
[![Stories in Ready](https://badge.waffle.io/compozed/deployadactyl.png?label=ready&title=Ready)](https://waffle.io/compozed/deployadactyl)
[![Gitter](https://badges.gitter.im/compozed/deployadactyl.svg)](https://gitter.im/compozed/deployadactyl?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
[![Slack Status](https://deployadactyl-invite.cfapps.io/badge.svg)](https://deployadactyl-invite.cfapps.io)
[![GoDoc](https://godoc.org/github.com/compozed/deployadactyl?status.svg)](https://godoc.org/github.com/compozed/deployadactyl)

Deployadactyl is a Go library for deploying applications to multiple [Cloud Foundry](https://www.cloudfoundry.org/) instances. Deployadactyl utilizes [blue green deployments](https://docs.pivotal.io/pivotalcf/devguide/deploy-apps/blue-green.html) and if it's unable to push an application it will rollback to the previous version. It also utilizes Go channels for concurrent deployments across the multiple Cloud Foundry instances.

Check out our stories on [Pivotal Tracker](https://www.pivotaltracker.com/n/projects/1912341)!

<!-- TOC depthFrom:2 depthTo:6 withLinks:1 updateOnSave:1 orderedList:0 -->

- [How It Works](#how-it-works)
- [Why Use Deployadactyl?](#why-use-deployadactyl)
- [Usage Requirements](#usage-requirements)
	- [Dependencies](#dependencies)
	- [Configuration File](#configuration-file)
		- [Example Configuration yml](#example-configuration-yml)
		- [Environment Variables](#environment-variables)
- [How to Download Dependencies](#how-to-download-dependencies)
- [How To Run Deployadactyl](#how-to-run-deployadactyl)
- [How to Push Deployadactyl to Cloud Foundry](#how-to-push-deployadactyl-to-cloud-foundry)
	- [Available Flags](#available-flags)
	- [API](#api)
		- [Example Curl](#example-curl)
- [Event Handling](#event-handling)
	- [Available Emitted Event Types](#available-emitted-event-types)
	- [Event Handler Example](#event-handler-example)
	- [Event Handling Example](#event-handling-example)
- [Contributing](#contributing)

<!-- /TOC -->

## How It Works

Deployadactyl works by utilizing the [Cloud Foundry CLI](http://docs.cloudfoundry.org/cf-cli/) to push an application. The general flow is to get a list of Cloud Foundry instances, check that the instances are available, download an artifact, log into each instance, and concurrently call `cf push` in the deploying applications directory. If an application fails to deploy on any instance, Deployadactyl will automatically roll the application back to the previous version.

## Why Use Deployadactyl?

As an application grows, it will have multiple foundations for each environment. These scaling foundations make deploying an application time consuming and difficult to manage. If any errors occur during a deployment it can greatly increase downtime.

Deployadactyl makes the process easy and efficient with:

- Management of multiple environment configurations
- Concurrent deployments across environment foundations
- Automatic rollbacks for failures or errors
- Prechecking foundation availablity before deployments
- Event handlers for third-party services


## Usage Requirements


### Dependencies

Deployadactyl has the following dependencies within the environment:

- [ CloudFoundry CLI](https://github.com/cloudfoundry/cli)
- [Go 1.6](https://golang.org/dl/) or later


### Configuration File

Deployadactyl needs a `yml` configuration file to specify environments to deploy to. Each environment has a name, domain and a list of foundations.

The configuration file can be placed anywhere within the Deployadactyl directory, or outside, as long as the location is specified when running the server.

|**Param**|**Necessity**|**Type**|**Description**|
|---|:---:|---|---|
|`name`|**Required**|`string`| Used in the deploy when the users are sending a request to Deployadactyl to specify which environment from the config they want to use.|
|`foundations` |**Required**|`[]string`|A list of Cloud Foundry Cloud Controller URLs.|
|`domain`|*Optional*|`string`| Used to specify a load balanced URL that has previously been created on the Cloud Foundry instances.|
|`authenticate` |*Optional*|`bool`| Used to specify if basic authentication is required for users. See the [authentication section](https://github.com/compozed/deployadactyl/wiki/Deployadactyl-API-v1.0.0#authentication) in the [API documentation](https://github.com/compozed/deployadactyl/wiki/Deployadactyl-API-Versions) for more details|
|`skip_ssl` |*Optional*|`bool`| Used to skip SSL verification when Deployadactyl logs into Cloud Foundry.|
|`instances` |*Optional*|`int`| Used to set the number of instances an application is deployed with. If the number of instances is specified in a Cloud Foundry manifest, that will be used instead. |

#### Example Configuration yml

```yaml
---
environments:
  - name: preproduction
    domain: preproduction.example.com
    foundations:
    - https://api.foundation-1.example.com
    - https://api.foundation-2.example.com
    authenticate: false
    skip_ssl: true
    instances: 2

  - name: production
    domain: production.example.com
    foundations:
    - https://production.foundation-1.example.com
    - https://production.foundation-2.example.com
    - https://production.foundation-3.example.com
    - https://production.foundation-4.example.com
    authenticate: true
    skip_ssl: false
    instances: 4
```

#### Environment Variables

Authentication is optional as long as `CF_USERNAME` and `CF_PASSWORD` environment variables are exported. We recommend making a generic user account that is able to push to each Cloud Foundry instance.

```bash
$ export CF_USERNAME=some-username
$ export CF_PASSWORD=some-password
```

*Optional:* The log level can be changed by defining `DEPLOYADACTYL_LOGLEVEL`. `DEBUG` is the default log level.

## How to Download Dependencies

We use [Godeps](https://github.com/tools/godep) to vendor our dependencies. To grab the dependencies and save them to the vendor folder, run the following commands:

```bash
$ go get -u github.com/tools/godep
$ rm -rf Godeps                      # this will clean the repo of it's dependencies
$ godep save ./...
```

or

```bash
$ make dependencies
```

## How To Run Deployadactyl

After a [configuration file](#configuration-file) has been created and environment variables have been set, the server can be run using the following commands:

```bash
$ cd ~/go/src/github.com/compozed/deployadactyl && go run server.go
```

or

```bash
$ cd ~/go/src/github.com/compozed/deployadactyl && go build && ./deployadactyl
```

## How to Push Deployadactyl to Cloud Foundry

To push Deployadactyl to Cloud Foundry, edit the `manifest.yml` to include the `CF_USERNAME` and `CF_PASSWORD` environment variables. In addition, be sure to create a `config.yml`. Then you can push to Cloud Foundry like normal:

```bash
$ cf login
$ cf push
```

or

```bash
$ make push
```

### Available Flags

|**Flag**|**Usage**|
|---|---|
|`-config`|location of the config file (default "./config.yml")
|`-envvar`|turns on the environment variable handler that will bind environment variables to your application at deploy time
|`-health-check`|turns on the health check handler that confirms an application is up and running before finishing a push
|`-route-mapper`|turns on the route mapper handler that will map additional routes to an application during a deployment. see the Cloud Foundry manifest documentation [here](https://docs.cloudfoundry.org/devguide/deploy-apps/manifest.html#routes) for more information

### API

A deployment by hitting the API using `curl` or other means. For more information on using the Deployadactyl API visit the [API documentation](https://github.com/compozed/deployadactyl/wiki/Deployadactyl-API-Versions) in the wiki.

#### Example Curl

```bash
curl -X POST \
     -u your_username:your_password \
     -H "Accept: application/json" \
     -H "Content-Type: application/json" \
     -d '{ "artifact_url": "https://example.com/lib/release/my_artifact.jar", "health_check_endpoint": "/health" }' \
     https://preproduction.example.com/v1/apps/environment/org/space/t-rex
```

## Event Handling

With Deployadactyl you can optionally register event handlers to perform any additional actions your deployment flow may require. For example, you may want to do an additional health check before the new application overwrites the old application.

### Available Events

|**Event Type**|**Returned Struct**|**Emitted**|
|---|---|---|
|`deploy.start`|[DeployEventData](structs/deploy_event_data.go)|Before deployment starts
|`deploy.success`|[DeployEventData](structs/deploy_event_data.go)|When a deployment succeeds
|`deploy.failure`|[DeployEventData](structs/deploy_event_data.go)|When a deployment fails
|`deploy.error`|[DeployEventData](structs/deploy_event_data.go)|When a deployment throws an error
|`deploy.finish`|[DeployEventData](structs/deploy_event_data.go)|When a deployment finishes, regardless of success or failure
|`push.finished`|[PushEventData](structs/push_event_data.go)| Happens before a push finishes. If it receives an error, it will stop the deployment and trigger an undo push
|`validate.foundationsUnavailable`|[PrecheckerEventData](structs/prechecker_event_data.go)|When a foundation you're deploying to is not running

### Event Handler Example

See the [Health Checker](eventmanager/handlers/healthchecker/healthchecker.go) for an example of how to write an event handler.

## Contributing

See our [CONTRUBUTING](CONTRIBUTING.md) section for more information.
