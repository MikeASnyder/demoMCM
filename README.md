#Rancher

Rancher is an open source project that provides a complete platform for operating Docker in production. It provides infrastructure services such as multi-host networking, global and local load balancing, and volume snapshots. It integrates native Docker management capabilities such as Docker Machine and Docker Swarm. It offers a rich user experience that enables devops admins to operate Docker in production at large scale.

## Installation

Rancher is deployed as a set of Docker containers.  Running Rancher is a simple as launching two containers.  One container as the management server and another container on a node as an agent.  You can install the containers in following approaches.

* [Manually](#launching-management-server)
* [Vagrant](#vagrant)
* [Puppet](https://github.com/nickschuch/puppet-rancher) (Thanks @nickschuch) 
* [Chef](https://github.com/paulczar/cookbook-rancher)

### Requirements

* Docker 1.6.0+
* Ubuntu 14.04 or CoreOS 494+
    * *Note: These are the only tested distributions at the moment, but most modern Linux distributions will work*
* RAM: 1GB+
 
### Launching Management Server

    docker run -d --restart=always -p 8080:8080 rancher/server

The UI and API are available on the exposed port `8080`.

### Using Rancher

To learn more about using Rancher, please refer to our [documentation](http://rancherio.github.io/rancher/). 
 
### Vagrant

If you want to use Vagrant to run this on your laptop just clone the repo and to do `vagrant up` and then access port 8080 for the UI.

## Source Code

This repo is a meta-repo used for packaging.  The source code for Rancher is in other repos in the rancherio organization.  The majority of the code is in https://github.com/rancherio/cattle.

## Support, Discussion, and Community
If you need any help with Rancher or RancherOS, please join us at either our [rancherio Google Groups](https://groups.google.com/forum/#!forum/rancherio) or [#rancher IRC channel](http://webchat.freenode.net/?channels=rancher) where most of our team hangs out at.

Please submit any **Rancher** bugs, issues, and feature requests to [rancherio/rancher](//github.com/rancherio/rancher/issues).

Please submit any **RancherOS** bugs, issues, and feature requests to [rancherio/os](//github.com/rancherio/os/issues).

#License
Copyright (c) 2014-2015 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

