<div align=center>
<h1><code>cogmoteGO</code></h1>
<b>"air traffic control" for remote neuroexperiments</b></br/>
</div>
<br/>

## Introduction

`cogmoteGO` is the "air traffic control" for remote neuroexperiments: a lightweight Go system coordinating distributed data streams, commands, and full experiment lifecycle management - from deployment to data collection.

## Bindings

- [for matlab](https://github.com/Ccccraz/matmoteGO.git)

## Installation

### For Linux & macOS

#### By install script

```sh
curl -sS https://raw.githubusercontent.com/Ccccraz/cogmoteGO/main/install.sh | sh
```

### For Windows

#### By install script

```sh
irm -Uri 'https://raw.githubusercontent.com/Ccccraz/cogmoteGO/main/install.ps1' | iex
```

#### By winget
> The winget version is relatively outdated; currently, we recommend installing via a script.

```sh
winget install ccccraz.cogmoteGO
```

## Getting started

### Run as service

#### For Linux & macOS

restart the service as user
> We recommend that you register cogmoteGO as a user service on Linux

```sh
cogmoteGO service -u
```

start the service as user

```sh
cogmoteGO service start -u
```

register the service

```sh
sudo cogmoteGO service
```

start the service

```sh
sudo cogmoteGO service start
```

#### For Windows

register the service

> note: you need to run the command as administrator

```sh
cogmoteGO service
```

start the service

```sh
cogmoteGO service start
```

restart the service as user

> note: the password is required for running the service as user

```sh
cogmoteGO service -u -p <your_password>
```

start the service as user

```sh
cogmoteGO service start -u
```

#### For all platforms

for more info about the service, run

```sh
cogmoteGO service --help
```


#### Test

```sh
curl --location --request GET 'http://localhost:9012/api/device'
```