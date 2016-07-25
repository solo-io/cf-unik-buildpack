# cf-unik-buildpack
CloudFoundry Buildpack for Building & Running Unikernels

To use:
In your `manifest.yml` (the CloudFoundry one), you must specify the Unik configuration like so:
```yaml
---
applications:
- name: myapp
env:
  URL: 52.12.10.128:3000
  PROVIDER: aws
  ARGS: optional string of args  
```

* URL: the url of the unik daemon
* PROVIDER: the cloud/vm provider to run on. Supported (as of 7/25/2016): AWS, Virtualbox, Vsphere
* ARGS: optional string of args to pass to the unikernel

Note:
* Supported languages (as of 7/25/2016): Java, Node.js, Python3, Golang

Run with
```
cf push -b=https://github.com/emc-advanced-dev/cf-unik-buildpack.git.git
```
