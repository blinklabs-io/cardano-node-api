# ApiUtxoItem

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | Pointer to **string** |  | [optional] 
**Amount** | Pointer to **int64** |  | [optional] 
**Assets** | Pointer to **map[string]interface{}** |  | [optional] 
**Index** | Pointer to **int32** |  | [optional] 
**TxHash** | Pointer to **string** |  | [optional] 

## Methods

### NewApiUtxoItem

`func NewApiUtxoItem() *ApiUtxoItem`

NewApiUtxoItem instantiates a new ApiUtxoItem object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewApiUtxoItemWithDefaults

`func NewApiUtxoItemWithDefaults() *ApiUtxoItem`

NewApiUtxoItemWithDefaults instantiates a new ApiUtxoItem object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddress

`func (o *ApiUtxoItem) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *ApiUtxoItem) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *ApiUtxoItem) SetAddress(v string)`

SetAddress sets Address field to given value.

### HasAddress

`func (o *ApiUtxoItem) HasAddress() bool`

HasAddress returns a boolean if a field has been set.

### GetAmount

`func (o *ApiUtxoItem) GetAmount() int64`

GetAmount returns the Amount field if non-nil, zero value otherwise.

### GetAmountOk

`func (o *ApiUtxoItem) GetAmountOk() (*int64, bool)`

GetAmountOk returns a tuple with the Amount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAmount

`func (o *ApiUtxoItem) SetAmount(v int64)`

SetAmount sets Amount field to given value.

### HasAmount

`func (o *ApiUtxoItem) HasAmount() bool`

HasAmount returns a boolean if a field has been set.

### GetAssets

`func (o *ApiUtxoItem) GetAssets() map[string]interface{}`

GetAssets returns the Assets field if non-nil, zero value otherwise.

### GetAssetsOk

`func (o *ApiUtxoItem) GetAssetsOk() (*map[string]interface{}, bool)`

GetAssetsOk returns a tuple with the Assets field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAssets

`func (o *ApiUtxoItem) SetAssets(v map[string]interface{})`

SetAssets sets Assets field to given value.

### HasAssets

`func (o *ApiUtxoItem) HasAssets() bool`

HasAssets returns a boolean if a field has been set.

### GetIndex

`func (o *ApiUtxoItem) GetIndex() int32`

GetIndex returns the Index field if non-nil, zero value otherwise.

### GetIndexOk

`func (o *ApiUtxoItem) GetIndexOk() (*int32, bool)`

GetIndexOk returns a tuple with the Index field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndex

`func (o *ApiUtxoItem) SetIndex(v int32)`

SetIndex sets Index field to given value.

### HasIndex

`func (o *ApiUtxoItem) HasIndex() bool`

HasIndex returns a boolean if a field has been set.

### GetTxHash

`func (o *ApiUtxoItem) GetTxHash() string`

GetTxHash returns the TxHash field if non-nil, zero value otherwise.

### GetTxHashOk

`func (o *ApiUtxoItem) GetTxHashOk() (*string, bool)`

GetTxHashOk returns a tuple with the TxHash field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTxHash

`func (o *ApiUtxoItem) SetTxHash(v string)`

SetTxHash sets TxHash field to given value.

### HasTxHash

`func (o *ApiUtxoItem) HasTxHash() bool`

HasTxHash returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


