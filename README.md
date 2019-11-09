![](https://raw.githubusercontent.com/compozed/images/master/deployadactyl_logo.png)

[![Release](https://img.shields.io/github/release/compozed/deployadactyl.svg)](https://github.com/compozed/deployadactyl/releases/latest)
[![CircleCI](https://circleci.com/gh/compozed/deployadactyl.svg?style=svg&circle-token=0eab8bce42440217fb24ffd8ffdc2b44932125d5)](https://circleci.com/gh/compozed/deployadactyl)
[![Go Report Card](https://goreportcard.com/badge/github.com/compozed/deployadactyl)](https://goreportcard.com/report/github.com/compozed/deployadactyl)
[![codecov](https://codecov.io/gh/compozed/deployadactyl/branch/master/graph/badge.svg?token=r9yd1cwtbH)](https://codecov.io/gh/compozed/deployadactyl)
[![Stories in Ready](https://badge.waffle.io/compozed/deployadactyl.png?label=ready&title=Ready)](https://waffle.io/compozed/deployadactyl)
[![Gitter](https://badges.gitter.im/compozed/deployadactyl.svg)](https://gitter.im/compozed/deployadactyl?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
[![GoDoc](https://godoc.org/github.com/compozed/deployadactyl?status.svg)](https://godoc.org/github.com/compozed/deployadactyl)

Deployadactyl is a Go library for managing applications across multiple [Cloud Foundry](https://www.cloudfoundry.org/) instances. Deployadactyl utilizes [blue green deployments](https://docs.pivotal.io/pivotalcf/devguide/deploy-apps/blue-green.html) and if it's unable to execute the requested operation it will rollback to the previous state. It also utilizes Go channels for concurrent deployments across the multiple Cloud Foundry instances.

Check out our stories on [Pivotal Tracker](https://www.pivotaltracker.com/n/projects/1912341)!

<!-- TOC depthFrom:2 depthTo:6 withLinks:1 updateOnSave:1 orderedList:0 -->

- [How It Works](#how-it-works)
- [Why Use Deployadactyl?](#why-use-deployadactyl)
- [Installation Requirements](#installation-requirements)
    - [Dependencies](#dependencies)
    - [Configuration File](#configuration-file)
        - [Example Configuration yml](#example-configuration-yml)
    - [Environment Variables](#environment-variables)
- [Installing Deployadactyl](#installing-deployadactyl)
    - [Local Installation](#local-installation)
    - [Cloud Foundry Installation](#cloud-foundry-installation)
    - [Available Flags](#available-flags)
- [API](#api)
    - [Example Push Curl](#example-push-curl)
    - [Example Stop Curl](#example-stop-curl)
- [Event Handling](#event-handling)
    - [Application Events](#application-events)
    - [Push Events](#push-events)
    - [Start Events](#start-events)
    - [Stop Events](#stop-events)
	- [Event Handler Example](#event-handler-example)
	- [Deprecated Event Handling](#deprecated-event-handling)
- [Contributing](#contributing)

<!-- /TOC -->

## How It Works

Deployadactyl works by utilizing the [Cloud Foundry CLI](http://docs.cloudfoundry.org/cf-cli/) to manage applications. The general flow is to get a list of Cloud Foundry instances, check that the instances are available, log into each instance, and concurrently execute the requested operation on each instance. If the requested operation fails, Deployadactyl will automatically revert the application back to the previous state.  For example, in the case of deploying an application, the specified artifact will be downloaded and `cf push` will be called concurrently in the deploying applications directory on each CF instance.  If the push fails on any instance, the application will be reverted to the version that was previously deployed on all instances.

## Why Use Deployadactyl?

As an application grows, it will have multiple foundations for each environment. These scaling foundations make managing an application time consuming and difficult to manage. Deployment errors can greatly increase downtime and result in inconsistent state of the application across all foundations..

Deployadactyl makes the process easy and efficient with:

- Management of multiple environment configurations
- Concurrent deployments and running state management across environment foundations
- Automatic rollbacks for failures or errors
- Prechecking foundation availablity before managing applicaiton state
- Event handlers for third-party services


## Installation Requirements


### Dependencies

Deployadactyl has the following dependencies within the environment:

- [ CloudFoundry CLI](https://github.com/cloudfoundry/cli)
- [Go 1.6](https://golang.org/dl/) or later


We use [Godeps](https://github.com/tools/godep) to vendor our GO dependencies. To grab the dependencies and save them to the vendor folder, run the following commands:

```bash
$ go get -u github.com/tools/godep
$ godep restore                       // updates local packages to required versions
$ rm -rf Godeps
$ godep save ./...                    // creates ./vendor folder with dependencies
```

or

```bash
$ make dependencies
```

or

If you have Go 1.11 or above you can use [Modules feature](https://github.com/golang/go/wiki/Modules) and skip above steps.

### Configuration File

Deployadactyl needs a `yml` configuration file to specify available environments for managing applications. At a minimum, each environment has a name and a list of foundations.

The configuration file can be placed anywhere within the Deployadactyl directory, or outside, as long as the location is specified when running the server.

|**Param**|**Necessity**|**Type**|**Description**|
|---|:---:|---|---|
|`name`|**Required**|`string`| Used in the deploy when the users are sending a request to Deployadactyl to specify which environment from the config they want to use.|
|`foundations` |**Required**|`[]string`|A list of Cloud Foundry Cloud Controller URLs.|
|`domain`|*Optional*|`string`| Used to specify a load balanced URL that has previously been created on the Cloud Foundry instances.|
|`authenticate` |*Optional*|`bool`| Used to specify if basic authentication is required for users. See the [authentication section](https://github.com/compozed/deployadactyl/wiki/Deployadactyl-API-v1.0.0#authentication) for more details|
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

### Environment Variables

Authentication is optional as long as `CF_USERNAME` and `CF_PASSWORD` environment variables are exported. We recommend making a generic user account that is able to push to each Cloud Foundry instance.

```bash
$ export CF_USERNAME=some-username
$ export CF_PASSWORD=some-password
```

*Optional:* The log level can be changed by defining `DEPLOYADACTYL_LOGLEVEL`. `DEBUG` is the default log level.

## Installing Deployadactyl

### Local Installation
After a [configuration file](#configuration-file) has been created and environment variables have been set, the server can be run using the following commands:

```bash
$ cd ~/go/src/github.com/compozed/deployadactyl && go run server.go
```

or

```bash
$ cd ~/go/src/github.com/compozed/deployadactyl && go build && ./deployadactyl
```

### Cloud Foundry Installation

To push Deployadactyl to Cloud Foundry, edit the `manifest.yml` to include the `CF_USERNAME` and `CF_PASSWORD` environment variables. In addition, be sure to create a `config.yml`. Then you can push to Cloud Foundry like normal:

```bash
$ cf login
$ cf push
```

or

```bash
$ make push
```

### Available Installation Flags

|**Flag**|**Usage**|
|---|---|
|`-config`|location of the config file (default "./config.yml")
|`-envvar`|turns on the environment variable handler that will bind environment variables to your application at deploy time
|`-health-check`|turns on the health check handler that confirms an application is up and running before finishing a push
|`-route-mapper`|turns on the route mapper handler that will map additional routes to an application during a deployment. see the Cloud Foundry manifest documentation [here](https://docs.cloudfoundry.org/devguide/deploy-apps/manifest.html#routes) for more information

## API

A deployment can be executed or modified by hitting the API using `curl` or other means. For more information on using the Deployadactyl API visit the [API documentation](https://github.com/compozed/deployadactyl/wiki) in the wiki.

### Example Push Curl

```bash
curl -X POST \
     -u your_username:your_password \
     -H "Accept: application/json" \
     -H "Content-Type: application/json" \
     -d '{ "artifact_url": "https://example.com/lib/release/my_artifact.jar", "health_check_endpoint": "/health" }' \
     https://preproduction.example.com/v3/deploy/environment/org/space/t-rex
```

### Example Stop Curl

```bash
curl -X PUT \
     -u your_username:your_password \
     -H "Accept: application/json" \
     -H "Content-Type: application/json" \
     -d '{ "state": "stopped" }' \
     https://preproduction.example.com/v3/deploy/environment/org/space/t-rex
```

## Event Handling

With Deployadactyl you can optionally register event handlers to perform any additional actions your deployment flow may require. For example, you may want to do an additional health check before the new application overwrites the old application.

***NOTE*** The event handling framework for Deployadactyl has been reworked in version 3 to allow for strongly typed binding between event handler functions and the events on which those functions operate.  See more info below and in the [wiki](https://github.com/compozed/deployadactyl/wiki/API-v3.0.0)


### Event Handler Example

Attach an event handler to a specific event by creating a binding between the desired event and your handler function and add it to the [EventManager](/eventmanager/eventmanager.go):

```
myHandler := func(event PushStartedEvent) error {
   mylog.Debug("A push has started with manifest: " + event.Manifest)
   ...
   return nil
}

eventManager.AddBinding(NewPushStartedEventBinding(myHandler))
```

Custom events can be created by implementing the [Binding](/interfaces/eventmanager.go) and [IEvent](/interfaces/eventmanager.go) interfaces.

### Deprecated Event Handling

Prior to version 3, events were registered the following way:

```
type Handler struct {...}

func (h Handler) OnEvent(event interfaces.Event) error {
   if event.Type == "push.started" {
      deploymentInfo := event.Data.(*DS.DeployEventData).DeploymentInfo
      mylog.Debug("A push has started with manifest: " + deploymentInfo.Manifest
      ...
      return nil
   } else ...
}

eventManager.AddHandler(Handler{...}, "push.started")
```

This method of event handling is still supported for push related events and creating custom events, but is deprecated and can be expected to be removed in the future.

## Contributing

See our [CONTRIBUTING](CONTRIBUTING.md) section for more information.
