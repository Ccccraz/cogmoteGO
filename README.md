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

### For Linux

```sh
curl -sS https://raw.githubusercontent.com/Ccccraz/cogmoteGO/main/install.sh | sh
```

### For Windows

```sh
winget install ccccraz.cogmoteGO
```

## Getting started

### Run as service

#### For Linux && MacOS

register the service

```sh
cogmoteGO service
```

start the service

```sh
cogmoteGO service start
```

#### For Windows

register the service

> note: you need to run the command as administrator

> note: the password is required

```sh
cogmoteGO service -p <your_password>
```

start the service

```sh
cogmoteGO service start
```

#### For all platforms

for more info about the service, run

```sh
cogmoteGO service --help
```