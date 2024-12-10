# \LocaltxsubmissionAPI

All URIs are relative to */api*

Method | HTTP request | Description
------------- | ------------- | -------------
[**LocaltxsubmissionTxPost**](LocaltxsubmissionAPI.md#LocaltxsubmissionTxPost) | **Post** /localtxsubmission/tx | Submit Tx



## LocaltxsubmissionTxPost

> string LocaltxsubmissionTxPost(ctx).ContentType(contentType).Execute()

Submit Tx



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
	contentType := "contentType_example" // string | Content type

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.LocaltxsubmissionAPI.LocaltxsubmissionTxPost(context.Background()).ContentType(contentType).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `LocaltxsubmissionAPI.LocaltxsubmissionTxPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `LocaltxsubmissionTxPost`: string
	fmt.Fprintf(os.Stdout, "Response from `LocaltxsubmissionAPI.LocaltxsubmissionTxPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiLocaltxsubmissionTxPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **contentType** | **string** | Content type | 

### Return type

**string**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

