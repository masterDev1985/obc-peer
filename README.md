# Openchain - Peer

## Overview

This project contains the core blockchain fabric.  

## Building the project

Assuming you have followed the [development environment getting started instructions](https://github.com/openblockchain/obc-getting-started/blob/master/devenv.md)

To access your VM, run
```
vagrant ssh
```

From within the VM, follow these additional steps.

### Go build
```
cd $GOPATH/src/github.com/openblockchain/obc-peer
go build
```

## Run

To see what commands are available, simply execute the following command:

    cd $GOPATH/src/github.com/openblockchain/obc-peer
    ./obc-peer

You should see some output similar to below (**NOTE**: rootcommand below is hardcoded in the [main.go](./main.go). Current build will actually create an *obc-peer* executable file).

```
    Usage:
      obc-peer [command]

    Available Commands:
      peer        Run obc peer.
      status      Status of the obc peer.
      stop        Stops the obc peer.
      chaincode    Compiles the specified chaincode.
      help        Help about any command

    Flags:
      -h, --help[=false]: help for openchain


    Use "obc-peer [command] --help" for more information about a command.
```

The **peer** command will run peer process. You can then use the other commands to interact with this peer process. For example, status will show the peer status.

## Test

To run all tests, in one window, run `./obc-peer peer`. In a second window

    cd $GOPATH/src/github.com/openblockchain/obc-peer
    go test -timeout=20m $(go list github.com/openblockchain/obc-peer/... | grep -v /vendor/)

Note that the first time the tests are run, they can take some time due to the need to download a docker image that is about 1GB in size. This is why the timeout flag is added to the above command.

To run a specific test use the `-run RE` flag where RE is a regular expression that matches the test name. To run tests with verbose output use the `-v` flag. For example, to run TestGetFoo function, change to the directory containing the `foo_test.go` and enter:

    go test -test.v -run=TestGetFoo

## Writing Chaincode
Since chaincode is written in Go language, you can set up the environment to accommodate the rapid edit-compile-run of your chaincode. Follow the instructions on the [Sandbox Setup](https://github.com/openblockchain/obc-peer/blob/master/docs/SandboxSetup.md) page, which allows you to run your chaincode off the blockchain.

## Setting Up a Network

To set up an Openchain network of several validating peers, follow the instructions on the [Devnet Setup](https://github.com/openblockchain/obc-peer/blob/master/docs/DevnetSetup.md)
page. This network leverage Docker to manage multiple instances of validating peer on the same machine, allowing you to quickly test your chaincode.


## Working with CLI, REST, and Node.js

When you are ready to start interacting with the Openchain peer node through the available APIs and packages, follow the instructions on the [API Documentation](https://github.com/openblockchain/obc-peer/blob/master/docs/Openchain%20API.md) page.

## Configuration

Configuration utilizes the [viper](https://github.com/spf13/viper) and [cobra](https://github.com/spf13/cobra) libraries.

There is an **openchain.yaml** file that contains the configuration for the peer process. Many of the configuration settings can be overridden at the command line by setting ENV variables that match the configuration setting, but by prefixing the tree with *'OPENCHAIN_'*. For example, logging level manipulation through the environment is shown below:

    OPENCHAIN_PEER_LOGGING_LEVEL=CRITICAL ./obc-peer

## Logging

Logging utilizes the [go-logging](https://github.com/op/go-logging) library.  

The available log levels in order of increasing verbosity are: *CRITICAL | ERROR | WARNING | NOTICE | INFO | DEBUG*

## Generating grpc code
If you modify ant .proto files, run the following command to generate new .pb.go files.
```
/openchain/obc-dev-env/compile_protos.sh
```

## Adding or updating a Go packages
Openchain uses the [Go 1.5 Vendor Experiment](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo/edit) for package management. This means that all required packages reside in the /vendor folder within the obc-peer project. This is enabled because the GO15VENDOREXPERIMENT environment variable is set to 1 in the Vagrant environment. Go will use packages in this folder instead of the GOPATH when `go install` or `go build` is run. To manage the packages in the /vendor folder, we use [Govendor](https://github.com/kardianos/govendor). This is installed in the Vagrant environment. The following commands can be used for package management.
```
# Add external packages.
govendor add +external

# Add a specific package.
govendor add github.com/kardianos/osext

# Update vendor packages.
govendor update +vendor

# Revert back to normal GOPATH packages.
govendor remove +vendor

# List package.
govendor list
```

## Building outside of Vagrant
This is not recommended, however some users may wish to build Openchain outside of Vagrant if they use an editor with built in Go tooling. The instructions are

1. Follow all steps required to setup and run a Vagrant image.
- Make you you have [Go 1.5.1](https://golang.org/) or later installed
- Set the GO15VENDOREXPERIMENT environmental variable to 1. `export GO15VENDOREXPERIMENT=1`
- Install [RocksDB](https://github.com/facebook/rocksdb/blob/master/INSTALL.md) version 4.1
- Run the following commands replacing `/opt/rocksdb` with the path where you installed RocksDB
```
cd $GOPATH/src/github.com/openblockchain/obc-peer
CGO_CFLAGS="-I/opt/rocksdb/include" CGO_LDFLAGS="-L/opt/rocksdb -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy" go install
```
