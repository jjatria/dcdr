# dcdr (decider)
Distributed Feature Flags for the Consul K/V Store

## Overview

Decider is a [feature flag](https://en.wikipedia.org/wiki/Feature_toggle) system built using the [Consul Key/Value Store](https://www.consul.io/intro/getting-started/kv.html). It supports both `percentile` and `boolean` flags for controlled infrastructure rollouts and kill switches. Included in this package is a CLI for modifying flags, a Golang client, and a HTTP server for accessing flags remotely.

Feature flags and remote configuration are hard problems to solve in the general sense. Most organizations will have many corner cases unique to their own infrastructure and policies that are cumbersome to solve in an abstract way. Decider is an extracted set of flexible libraries that we at VSCO have developed over the past year that have worked well for us in solving these problems. 

This package does not set out to solve problems like authentication or ACLs for your features but It does aim provide enough of the tooling and libraries so that you can do so yourself.

### Consul Integration
Decider uses the built in commands from the [Consul](http://consul.io) CLI to distribute feature flags throughout your cluster. All Consul configuration environment variables are used to ensure that Decider can be used anywhere a `consul agent` can be run. Similar to the concepts introduced by [`consul-template`](https://github.com/hashicorp/consul-template). Decider observes a key prefix in the store and then writes the resulting key/value tree to a flat JSON file on any node running the `dcdr watch` command. Clients then observe this file using `fsnotify` and reload their internal feature maps accordingly.

### Scopes
In order to allow for expanding use cases and to avoid naming collisions, Decider provides arbitrary scoping of feature flags. An example use case would be providing separate features sets according to country code or mobile platform. Additionally, multiple Decider instances can be run within a cluster with separate namespaces and key sets by configuring `/etc/dcdr/config.hcl`.

### Audit Trail
Due to the sensitive nature of configuration management, knowing the who, what, and when of changes can be very important. Decider uses `git` to handle this responsibility. By easily specifying a `git` repository and its origin in `/etc/dcdr/config.hcl`, Decider will export your keyspace as a `JSON` file and then commit and push the changeset to the specified origin. Of course, this is all optional if you enjoy living dangerously.

![](./resources/repo.png)

### Statsd Integration
It's nice to know when changes are happening. Decider can be configured to emit [Statsd](https://github.com/etsy/statsd) events when changes occur. Custom event tags can be sent as well if your collector supports them. Included in this package is a [DataDog](https://www.datadoghq.com/) adapter with Event and Tag support.

## Installation

* Install via `go get`
	* 	`go get -a github.com/vsco/dcdr`
* Install via release
	*  [https://github.com/vsco/dcdr/releases](https://github.com/vsco/dcdr/releases)
* Build from source

```
git clone git@github.com:vsco/dcdr.git
cd dcdr
script/bootstrap
script/install
```

Once installed on a machine running a `consul agent`, Decider is ready to connect to a default Consul host and port (localhost:8500) and begin writing to the K/V store. If you have a custom configuration for your agent, Decider will use the same [environment variables](https://github.com/hashicorp/consul/blob/master/api/api.go#L126) used to configure the Consul CLI.

## Getting Started

### CLI
The `dcdr` CLI has comprehensive help system for all commands.

```bash
dcdr help [command]" for more information about a command.
```

### Setting Features

Features have several fields that are accessible via `set` command.

```bash
	-n, --name="flag_name"
		the name of the flag to set
	-v, --value=0.0-1.0 or true|false
		the value of the flag
	-c, --comment="flag description"
		an optional comment or description
	-s, --scope="users/beta"
		an optional scope to nest the flag within
```

#### Example

```bash
dcdr set -n new-feature -v 0.1 -c "some new feature" -s user-groups/beta
```

The above command sets the key `dcdr/features/user-groups/beta/new-feature` equal to `0.1` and commits the update to the audit repo.

![](./resources/set.png)

### Listing Features

Listing features can be filtered by a given `scope` and `prefix`. Any further fanciness can be handled by piping the output to `grep` or `less`.

```bash
	-p, --prefix="<flag-prefix>"
		List only flags with matching prefix.
	-s, --scope="<flag-scope>"
		List only flags within a scope.
```

#### Example

```bash
dcdr list -p new -s user-groups/beta
```

![](./resources/list.png)

### Deleting Features

Features are removed using the `dcdr delete` command and take a `name` and `scope` parameters. If no `scope` is provided the `default` scope is assumed. Once deleted and if you have a repository configured, Decider will commit the changeset and push it to origin.

```
	-p, --prefix="<flag-prefix>"
		Name of the flag to delete
	-s, --scope="<flag-scope>"
		Optional scope to delete the flag from
```

#### Example

```
dcdr delete -n another-feature -s user-groups/beta
```

![](./resources/delete.png)

### Starting the Watcher

The `watch` command is central to how Decider features are distributed to nodes in a cluster. This command is a wrapper around `consul watch`. It observes the configured keyspace and writes a `JSON` file containing the exported structure to the configured [`Server:OutputPath`](https://github.com/vsco/dcdr/blob/readme-updates/config/config.go#L29).

If this path does not exist you will need to create it.

```
 sudo mkdir /etc/dcdr
 sudo chown `whoami` /etc/dcdr
```

![](./resources/watch.png)

#### Tying it together
If you need instructions for getting Consul installed, check their [Getting Started](https://www.consul.io/intro/getting-started/install.html) page.

Let's start a `consul agent` with an empty feature set and see how this all works together. For simplicity we can use the default Decider configuration without a git repository for auditing.

```
consul agent -bind "127.0.0.1" -dev
```

This will start a local Consul agent ready to accept connections on `http://127.0.0.1:8500`. Decider should now be able to connect to this instance and set features.

####Set some features

```
# check that we can talk to the local agent
~  → dcdr list
[dcdr] no feature flags found in namespace: dcdr

# set a feature into the 'default' scope.
~  → dcdr set -n example-feature -v false
[dcdr] set flag 'dcdr/features/default/example-feature'

# set a feature into the 'user-groups/beta' scope.
~  → dcdr set -n example-feature -v true -s user-groups/beta
[dcdr] set flag 'dcdr/features/user-groups/beta/example-feature'

~  → dcdr list

Name             Type     Value  Comment  Scope             Updated By
example-feature  boolean  false           default           chrisb
example-feature  boolean  true            user-groups/beta  chrisb
```
Here we have set the feature `example-feature` into two separate scopes. In the 'default' scope the value is `false` and in the 'user-groups/beta' scope it has been set to true.

##### Start the watcher and observe changes

```
# start the watcher
~  → dcdr watch
[dcdr] 2016/03/09 17:56:17.250948 watching namespace: dcdr
[dcdr] 2016/03/09 17:56:17.365362 wrote changes to /etc/dcdr/decider.json
```
The watcher is now observing your keyspace and writing all changes to the [`Server:OutputPath`](https://github.com/vsco/dcdr/blob/readme-updates/config/config.go#L29) (`/etc/dcdr/decider.json`).

The easiest way to view your feature flags is with `dcdr server`. This is a bare bones implementation of how to access features over HTTP. There is no authentication, so unless your use case is for internal access only you should include the `server` package into a new project and assemble your own. [EXAMPLE COMING, see server/demo/main.go for now].

```
# start the server
~  → dcdr server
[dcdr] started watching /etc/dcdr/decider.json
[dcdr] 2016/03/09 18:03:46.211150 serving /dcdr.json on :8000
```
The server is now running on `:8000` and features can be accessed by curling `:8000/dcdr.json`. In order to access your scopes the server accepts a `x-dcdr-scopes` header. This is a comma-delimited, priority-ordered list of scopes. Meaning that the scopes should provided with the highest priority first. For now we only have one scope so let's start simple.

```
# curl with no scopes
~  → curl -s :8000/dcdr.json
{
  "info": {},
  "dcdr": {
    "example-feature": false
  }
}
```
Here we see that the default value of false is returned. The `info` key is where information  like the current SHA of the repository would be if one was configured. Next, if we add the scope header we can access our scoped values.

```
~  → curl -sH "x-dcdr-scopes: user-groups/beta" :8000/dcdr.json
{
  "dcdr": {
    "info": {},
    "features": {
      "example-feature": true
    }
  }
}
```

### Using the Go client

Included in this package is a Go client. By default this client uses the same `config.hcl` for its configuration. You may also provide custom your own custom configuration as well. For this example we will assume the defaults are still in place and that the feature from the above example have been set.

#### Require and initialize the client

```Go
import "github.com/vsco/dcdr/client"

// Initialize a client with the default configuration
client := client.NewDefault()

// Begin watching the decider.json file
err := client.Watch()

if err != nil {
	panic(err)
}
```

#### Checking feature flags

The client has two main methods for interacting with flags `IsAvailable(feature string)` and `IsAvailableForId(feature string, id uint64)`.

**IsAvailable**

This method is for checking `boolean` features or 'kill switches'. If a `percentile` feature is passed to this method it will always return false. So don't do that.

```
# set the default feature
dcdr set -n example-feature -v false
```

```Go
// Using the fluent API for brevity
client, err := client.NewDefault().Watch()

if err != nil {
	panic(err)
}

// example-feature would be false
if client.IsAvailable("example-feature") {
	fmt.Println("example-feature enabled")
} else {
	fmt.Println("example-feature disabled")
}
```

This example initializes a new `Client` and begins watching the 'default' feature scope. It then checks the `example-feature` and runs the appropriate path given the current return value.

**So what about scopes?**

To initialize a Decider `Client` into a given set of scopes use the `WithScopes(scopes ...string)` method. This method creates a new `Client` with an underlying feature set that has the provided `scope` values merged onto the default set. If a feature does not exist in any of the provided scopes the client will fallback to the 'default' `scope`. This provides a mechanism for overriding features in a priority order.

```
# set the scoped feature
dcdr set -n example-feature -v true -s user-groups/beta
```

```Go
client, err := client.NewDefault().Watch()
scopedClient := client.WithScopes("user-groups/beta")

if err != nil {
	panic(err)
}

// example-feature would be true
if scopedClient.IsAvailable("example-feature") {
	fmt.Println("example-feature enabled")
} else {
	fmt.Println("example-feature disabled")
}
```

**Fallbacks**

```
# set a feature that does not exist in user-groups/beta
dcdr set -n another-feature -v true
```

```Go
client, err := client.NewDefault().Watch()
scopedClient := client.WithScopes("user-groups/beta")

if err != nil {
	panic(err)
}

// another-feature would be true
if scopedClient.IsAvailable("another-feature") {
	fmt.Println("another-feature enabled")
} else {
	fmt.Println("another-feature disabled")
}
```

**IsAvailableForId**

This method method works exactly as `IsAvailable` except that it is used for enabling features for only a fraction of requests. Both the `feature` string and `id` are hashed together using `hash/crc32` to create an integer that is used with the `float64` value of a `percentile` feature to determine the enabled state.

**Using percentiles**

```
# set a feature to 50%
dcdr set -n new-feature-rollout -v 0.5
```

```Go
client, err := client.NewDefault().Watch()

if err != nil {
	panic(err)
}

userId := unint64(5)

// new-feature-rollout would be true
if client.IsAvailableForId("new-feature-rollout", userId) {
	fmt.Println("new-feature-rollout enabled")
} else {
	fmt.Println("new-feature-rollout disabled")
}
```

### Configuration

All configuration lives in `/etc/dcdr/config.hcl`. You will need to create the `/etc/dcdr` directory and your permissions may differ but to get started locally do the following. 

```
 sudo mkdir /etc/dcdr
 sudo chown `whoami` /etc/dcdr
 dcdr init
```

`dcdr init` will create the default config file for you. Here is the example config.

```
Username = "twosim"
Namespace = "dcdr"

Watcher {
  OutputPath = "/etc/dcdr/decider.json"
}

Server {
  JsonRoot = "dcdr"
  Endpoint = "/dcdr.json"
}

Git {
  RepoURL = "git@github.com:vsco/decider-test-config.git"
  RepoPath = "/etc/dcdr/audit"
}

Stats {
  Namespace = "decider"
  Host = "127.0.0.1"
  Port = 8126
}
```




