# \ChainsyncAPI

All URIs are relative to *http://localhost/api*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ChainsyncSyncGet**](ChainsyncAPI.md#ChainsyncSyncGet) | **Get** /chainsync/sync | Start a chain-sync using a websocket for events



## ChainsyncSyncGet

> ChainsyncSyncGet(ctx).Tip(tip).Slot(slot).Hash(hash).Execute()

Start a chain-sync using a websocket for events

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
	tip := true // bool | whether to start from the current tip (optional)
	slot := int32(56) // int32 | slot to start sync at, should match hash (optional)
	hash := "hash_example" // string | block hash to start sync at, should match slot (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.ChainsyncAPI.ChainsyncSyncGet(context.Background()).Tip(tip).Slot(slot).Hash(hash).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ChainsyncAPI.ChainsyncSyncGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiChainsyncSyncGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tip** | **bool** | whether to start from the current tip | 
 **slot** | **int32** | slot to start sync at, should match hash | 
 **hash** | **string** | block hash to start sync at, should match slot | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: */*

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

