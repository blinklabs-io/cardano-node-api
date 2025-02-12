# Go API client for openapi

Cardano Node API

## Overview
This API client was generated by the [OpenAPI Generator](https://openapi-generator.tech) project.  By using the [OpenAPI-spec](https://www.openapis.org/) from a remote server, you can easily generate an API client.

- API version: 1.0
- Package version: 1.0.0
- Generator version: 7.11.0-SNAPSHOT
- Build package: org.openapitools.codegen.languages.GoClientCodegen
For more information, please visit [https://blinklabs.io](https://blinklabs.io)

## Installation

Install the following dependencies:

```sh
go get github.com/stretchr/testify/assert
go get golang.org/x/net/context
```

Put the package under your project folder and add the following in import:

```go
import openapi "github.com/blinklabs-io/cardano-node-api/openapi"
```

To use a proxy, set the environment variable `HTTP_PROXY`:

```go
os.Setenv("HTTP_PROXY", "http://proxy_name:proxy_port")
```

## Configuration of Server URL

Default configuration comes with `Servers` field that contains server objects as defined in the OpenAPI specification.

### Select Server Configuration

For using other server than the one defined on index 0 set context value `openapi.ContextServerIndex` of type `int`.

```go
ctx := context.WithValue(context.Background(), openapi.ContextServerIndex, 1)
```

### Templated Server URL

Templated server URL is formatted using default variables from configuration or from context value `openapi.ContextServerVariables` of type `map[string]string`.

```go
ctx := context.WithValue(context.Background(), openapi.ContextServerVariables, map[string]string{
	"basePath": "v2",
})
```

Note, enum values are always validated and all unused variables are silently ignored.

### URLs Configuration per Operation

Each operation can use different server URL defined using `OperationServers` map in the `Configuration`.
An operation is uniquely identified by `"{classname}Service.{nickname}"` string.
Similar rules for overriding default operation server index and variables applies by using `openapi.ContextOperationServerIndices` and `openapi.ContextOperationServerVariables` context maps.

```go
ctx := context.WithValue(context.Background(), openapi.ContextOperationServerIndices, map[string]int{
	"{classname}Service.{nickname}": 2,
})
ctx = context.WithValue(context.Background(), openapi.ContextOperationServerVariables, map[string]map[string]string{
	"{classname}Service.{nickname}": {
		"port": "8443",
	},
})
```

## Documentation for API Endpoints

All URIs are relative to */api*

Class | Method | HTTP request | Description
------------ | ------------- | ------------- | -------------
*ChainsyncAPI* | [**ChainsyncSyncGet**](docs/ChainsyncAPI.md#chainsyncsyncget) | **Get** /chainsync/sync | Start a chain-sync using a websocket for events
*LocalstatequeryAPI* | [**LocalstatequeryCurrentEraGet**](docs/LocalstatequeryAPI.md#localstatequerycurrenteraget) | **Get** /localstatequery/current-era | Query Current Era
*LocalstatequeryAPI* | [**LocalstatequeryEraHistoryGet**](docs/LocalstatequeryAPI.md#localstatequeryerahistoryget) | **Get** /localstatequery/era-history | Query Era History
*LocalstatequeryAPI* | [**LocalstatequeryGenesisConfigGet**](docs/LocalstatequeryAPI.md#localstatequerygenesisconfigget) | **Get** /localstatequery/genesis-config | Query Genesis Config
*LocalstatequeryAPI* | [**LocalstatequeryProtocolParamsGet**](docs/LocalstatequeryAPI.md#localstatequeryprotocolparamsget) | **Get** /localstatequery/protocol-params | Query Current Protocol Parameters
*LocalstatequeryAPI* | [**LocalstatequerySystemStartGet**](docs/LocalstatequeryAPI.md#localstatequerysystemstartget) | **Get** /localstatequery/system-start | Query System Start
*LocalstatequeryAPI* | [**LocalstatequeryTipGet**](docs/LocalstatequeryAPI.md#localstatequerytipget) | **Get** /localstatequery/tip | Query Chain Tip
*LocaltxmonitorAPI* | [**LocaltxmonitorHasTxTxHashGet**](docs/LocaltxmonitorAPI.md#localtxmonitorhastxtxhashget) | **Get** /localtxmonitor/has_tx/{tx_hash} | Check if a particular TX exists in the mempool
*LocaltxmonitorAPI* | [**LocaltxmonitorSizesGet**](docs/LocaltxmonitorAPI.md#localtxmonitorsizesget) | **Get** /localtxmonitor/sizes | Get mempool capacity, size, and TX count
*LocaltxmonitorAPI* | [**LocaltxmonitorTxsGet**](docs/LocaltxmonitorAPI.md#localtxmonitortxsget) | **Get** /localtxmonitor/txs | List all transactions in the mempool
*LocaltxsubmissionAPI* | [**LocaltxsubmissionTxPost**](docs/LocaltxsubmissionAPI.md#localtxsubmissiontxpost) | **Post** /localtxsubmission/tx | Submit Tx


## Documentation For Models

 - [ApiResponseApiError](docs/ApiResponseApiError.md)
 - [ApiResponseLocalStateQueryCurrentEra](docs/ApiResponseLocalStateQueryCurrentEra.md)
 - [ApiResponseLocalStateQuerySystemStart](docs/ApiResponseLocalStateQuerySystemStart.md)
 - [ApiResponseLocalStateQueryTip](docs/ApiResponseLocalStateQueryTip.md)
 - [ApiResponseLocalTxMonitorHasTx](docs/ApiResponseLocalTxMonitorHasTx.md)
 - [ApiResponseLocalTxMonitorSizes](docs/ApiResponseLocalTxMonitorSizes.md)
 - [ApiResponseLocalTxMonitorTxs](docs/ApiResponseLocalTxMonitorTxs.md)


## Documentation For Authorization

Endpoints do not require authorization.


## Documentation for Utility Methods

Due to the fact that model structure members are all pointers, this package contains
a number of utility functions to easily obtain pointers to values of basic types.
Each of these functions takes a value of the given basic type and returns a pointer to it:

* `PtrBool`
* `PtrInt`
* `PtrInt32`
* `PtrInt64`
* `PtrFloat`
* `PtrFloat32`
* `PtrFloat64`
* `PtrString`
* `PtrTime`

## Author

support@blinklabs.io

