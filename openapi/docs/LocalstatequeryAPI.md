# \LocalstatequeryAPI

All URIs are relative to *http://localhost/api*

Method | HTTP request | Description
------------- | ------------- | -------------
[**LocalstatequeryCurrentEraGet**](LocalstatequeryAPI.md#LocalstatequeryCurrentEraGet) | **Get** /localstatequery/current-era | Query Current Era
[**LocalstatequeryEraHistoryGet**](LocalstatequeryAPI.md#LocalstatequeryEraHistoryGet) | **Get** /localstatequery/era-history | Query Era History
[**LocalstatequeryGenesisConfigGet**](LocalstatequeryAPI.md#LocalstatequeryGenesisConfigGet) | **Get** /localstatequery/genesis-config | Query Genesis Config
[**LocalstatequeryProtocolParamsGet**](LocalstatequeryAPI.md#LocalstatequeryProtocolParamsGet) | **Get** /localstatequery/protocol-params | Query Current Protocol Parameters
[**LocalstatequerySystemStartGet**](LocalstatequeryAPI.md#LocalstatequerySystemStartGet) | **Get** /localstatequery/system-start | Query System Start
[**LocalstatequeryTipGet**](LocalstatequeryAPI.md#LocalstatequeryTipGet) | **Get** /localstatequery/tip | Query Chain Tip



## LocalstatequeryCurrentEraGet

> ApiResponseLocalStateQueryCurrentEra LocalstatequeryCurrentEraGet(ctx).Execute()

Query Current Era

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
	resp, r, err := apiClient.LocalstatequeryAPI.LocalstatequeryCurrentEraGet(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `LocalstatequeryAPI.LocalstatequeryCurrentEraGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `LocalstatequeryCurrentEraGet`: ApiResponseLocalStateQueryCurrentEra
	fmt.Fprintf(os.Stdout, "Response from `LocalstatequeryAPI.LocalstatequeryCurrentEraGet`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiLocalstatequeryCurrentEraGetRequest struct via the builder pattern


### Return type

[**ApiResponseLocalStateQueryCurrentEra**](ApiResponseLocalStateQueryCurrentEra.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## LocalstatequeryEraHistoryGet

> map[string]interface{} LocalstatequeryEraHistoryGet(ctx).Execute()

Query Era History

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
	resp, r, err := apiClient.LocalstatequeryAPI.LocalstatequeryEraHistoryGet(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `LocalstatequeryAPI.LocalstatequeryEraHistoryGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `LocalstatequeryEraHistoryGet`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `LocalstatequeryAPI.LocalstatequeryEraHistoryGet`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiLocalstatequeryEraHistoryGetRequest struct via the builder pattern


### Return type

**map[string]interface{}**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## LocalstatequeryGenesisConfigGet

> map[string]interface{} LocalstatequeryGenesisConfigGet(ctx).Execute()

Query Genesis Config

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
	resp, r, err := apiClient.LocalstatequeryAPI.LocalstatequeryGenesisConfigGet(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `LocalstatequeryAPI.LocalstatequeryGenesisConfigGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `LocalstatequeryGenesisConfigGet`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `LocalstatequeryAPI.LocalstatequeryGenesisConfigGet`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiLocalstatequeryGenesisConfigGetRequest struct via the builder pattern


### Return type

**map[string]interface{}**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## LocalstatequeryProtocolParamsGet

> map[string]interface{} LocalstatequeryProtocolParamsGet(ctx).Execute()

Query Current Protocol Parameters

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
	resp, r, err := apiClient.LocalstatequeryAPI.LocalstatequeryProtocolParamsGet(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `LocalstatequeryAPI.LocalstatequeryProtocolParamsGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `LocalstatequeryProtocolParamsGet`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `LocalstatequeryAPI.LocalstatequeryProtocolParamsGet`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiLocalstatequeryProtocolParamsGetRequest struct via the builder pattern


### Return type

**map[string]interface{}**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## LocalstatequerySystemStartGet

> ApiResponseLocalStateQuerySystemStart LocalstatequerySystemStartGet(ctx).Execute()

Query System Start

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
	resp, r, err := apiClient.LocalstatequeryAPI.LocalstatequerySystemStartGet(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `LocalstatequeryAPI.LocalstatequerySystemStartGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `LocalstatequerySystemStartGet`: ApiResponseLocalStateQuerySystemStart
	fmt.Fprintf(os.Stdout, "Response from `LocalstatequeryAPI.LocalstatequerySystemStartGet`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiLocalstatequerySystemStartGetRequest struct via the builder pattern


### Return type

[**ApiResponseLocalStateQuerySystemStart**](ApiResponseLocalStateQuerySystemStart.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## LocalstatequeryTipGet

> ApiResponseLocalStateQueryTip LocalstatequeryTipGet(ctx).Execute()

Query Chain Tip

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
	resp, r, err := apiClient.LocalstatequeryAPI.LocalstatequeryTipGet(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `LocalstatequeryAPI.LocalstatequeryTipGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `LocalstatequeryTipGet`: ApiResponseLocalStateQueryTip
	fmt.Fprintf(os.Stdout, "Response from `LocalstatequeryAPI.LocalstatequeryTipGet`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiLocalstatequeryTipGetRequest struct via the builder pattern


### Return type

[**ApiResponseLocalStateQueryTip**](ApiResponseLocalStateQueryTip.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

