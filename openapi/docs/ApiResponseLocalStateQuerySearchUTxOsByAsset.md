# ApiResponseLocalStateQuerySearchUTxOsByAsset

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Count** | Pointer to **int32** |  | [optional] 
**Utxos** | Pointer to [**[]ApiUtxoItem**](ApiUtxoItem.md) |  | [optional] 

## Methods

### NewApiResponseLocalStateQuerySearchUTxOsByAsset

`func NewApiResponseLocalStateQuerySearchUTxOsByAsset() *ApiResponseLocalStateQuerySearchUTxOsByAsset`

NewApiResponseLocalStateQuerySearchUTxOsByAsset instantiates a new ApiResponseLocalStateQuerySearchUTxOsByAsset object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewApiResponseLocalStateQuerySearchUTxOsByAssetWithDefaults

`func NewApiResponseLocalStateQuerySearchUTxOsByAssetWithDefaults() *ApiResponseLocalStateQuerySearchUTxOsByAsset`

NewApiResponseLocalStateQuerySearchUTxOsByAssetWithDefaults instantiates a new ApiResponseLocalStateQuerySearchUTxOsByAsset object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCount

`func (o *ApiResponseLocalStateQuerySearchUTxOsByAsset) GetCount() int32`

GetCount returns the Count field if non-nil, zero value otherwise.

### GetCountOk

`func (o *ApiResponseLocalStateQuerySearchUTxOsByAsset) GetCountOk() (*int32, bool)`

GetCountOk returns a tuple with the Count field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCount

`func (o *ApiResponseLocalStateQuerySearchUTxOsByAsset) SetCount(v int32)`

SetCount sets Count field to given value.

### HasCount

`func (o *ApiResponseLocalStateQuerySearchUTxOsByAsset) HasCount() bool`

HasCount returns a boolean if a field has been set.

### GetUtxos

`func (o *ApiResponseLocalStateQuerySearchUTxOsByAsset) GetUtxos() []ApiUtxoItem`

GetUtxos returns the Utxos field if non-nil, zero value otherwise.

### GetUtxosOk

`func (o *ApiResponseLocalStateQuerySearchUTxOsByAsset) GetUtxosOk() (*[]ApiUtxoItem, bool)`

GetUtxosOk returns a tuple with the Utxos field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUtxos

`func (o *ApiResponseLocalStateQuerySearchUTxOsByAsset) SetUtxos(v []ApiUtxoItem)`

SetUtxos sets Utxos field to given value.

### HasUtxos

`func (o *ApiResponseLocalStateQuerySearchUTxOsByAsset) HasUtxos() bool`

HasUtxos returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


