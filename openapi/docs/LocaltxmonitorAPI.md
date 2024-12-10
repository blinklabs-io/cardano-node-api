# \LocaltxmonitorAPI

All URIs are relative to */api*

Method | HTTP request | Description
------------- | ------------- | -------------
[**LocaltxmonitorHasTxTxHashGet**](LocaltxmonitorAPI.md#LocaltxmonitorHasTxTxHashGet) | **Get** /localtxmonitor/has_tx/{tx_hash} | Check if a particular TX exists in the mempool
[**LocaltxmonitorSizesGet**](LocaltxmonitorAPI.md#LocaltxmonitorSizesGet) | **Get** /localtxmonitor/sizes | Get mempool capacity, size, and TX count
[**LocaltxmonitorTxsGet**](LocaltxmonitorAPI.md#LocaltxmonitorTxsGet) | **Get** /localtxmonitor/txs | List all transactions in the mempool



## LocaltxmonitorHasTxTxHashGet

> ApiResponseLocalTxMonitorHasTx LocaltxmonitorHasTxTxHashGet(ctx, txHash).Execute()

Check if a particular TX exists in the mempool

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/blinklabs-io/cardano-node-api/openapi"
)

func main() {
	txHash := "txHash_example" // string | Transaction hash (hex string)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.LocaltxmonitorAPI.LocaltxmonitorHasTxTxHashGet(context.Background(), txHash).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `LocaltxmonitorAPI.LocaltxmonitorHasTxTxHashGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `LocaltxmonitorHasTxTxHashGet`: ApiResponseLocalTxMonitorHasTx
	fmt.Fprintf(os.Stdout, "Response from `LocaltxmonitorAPI.LocaltxmonitorHasTxTxHashGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**txHash** | **string** | Transaction hash (hex string) | 

### Other Parameters

Other parameters are passed through a pointer to a apiLocaltxmonitorHasTxTxHashGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**ApiResponseLocalTxMonitorHasTx**](ApiResponseLocalTxMonitorHasTx.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## LocaltxmonitorSizesGet

> ApiResponseLocalTxMonitorSizes LocaltxmonitorSizesGet(ctx).Execute()

Get mempool capacity, size, and TX count

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/blinklabs-io/cardano-node-api/openapi"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.LocaltxmonitorAPI.LocaltxmonitorSizesGet(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `LocaltxmonitorAPI.LocaltxmonitorSizesGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `LocaltxmonitorSizesGet`: ApiResponseLocalTxMonitorSizes
	fmt.Fprintf(os.Stdout, "Response from `LocaltxmonitorAPI.LocaltxmonitorSizesGet`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiLocaltxmonitorSizesGetRequest struct via the builder pattern


### Return type

[**ApiResponseLocalTxMonitorSizes**](ApiResponseLocalTxMonitorSizes.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## LocaltxmonitorTxsGet

> []ApiResponseLocalTxMonitorTxs LocaltxmonitorTxsGet(ctx).Execute()

List all transactions in the mempool

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/blinklabs-io/cardano-node-api/openapi"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.LocaltxmonitorAPI.LocaltxmonitorTxsGet(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `LocaltxmonitorAPI.LocaltxmonitorTxsGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `LocaltxmonitorTxsGet`: []ApiResponseLocalTxMonitorTxs
	fmt.Fprintf(os.Stdout, "Response from `LocaltxmonitorAPI.LocaltxmonitorTxsGet`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiLocaltxmonitorTxsGetRequest struct via the builder pattern


### Return type

[**[]ApiResponseLocalTxMonitorTxs**](ApiResponseLocalTxMonitorTxs.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

