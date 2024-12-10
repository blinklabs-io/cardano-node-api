/*
cardano-node-api

Cardano Node API

API version: 1.0
Contact: support@blinklabs.io
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package openapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
)

type LocaltxsubmissionAPI interface {

	/*
		LocaltxsubmissionTxPost Submit Tx

		Submit an already serialized transaction to the network.

		@param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
		@return LocaltxsubmissionAPILocaltxsubmissionTxPostRequest
	*/
	LocaltxsubmissionTxPost(
		ctx context.Context,
	) LocaltxsubmissionAPILocaltxsubmissionTxPostRequest

	// LocaltxsubmissionTxPostExecute executes the request
	//  @return string
	LocaltxsubmissionTxPostExecute(
		r LocaltxsubmissionAPILocaltxsubmissionTxPostRequest,
	) (string, *http.Response, error)
}

// LocaltxsubmissionAPIService LocaltxsubmissionAPI service
type LocaltxsubmissionAPIService service

type LocaltxsubmissionAPILocaltxsubmissionTxPostRequest struct {
	ctx         context.Context
	ApiService  LocaltxsubmissionAPI
	contentType *string
}

// Content type
func (r LocaltxsubmissionAPILocaltxsubmissionTxPostRequest) ContentType(
	contentType string,
) LocaltxsubmissionAPILocaltxsubmissionTxPostRequest {
	r.contentType = &contentType
	return r
}

func (r LocaltxsubmissionAPILocaltxsubmissionTxPostRequest) Execute() (string, *http.Response, error) {
	return r.ApiService.LocaltxsubmissionTxPostExecute(r)
}

/*
LocaltxsubmissionTxPost Submit Tx

Submit an already serialized transaction to the network.

	@param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
	@return LocaltxsubmissionAPILocaltxsubmissionTxPostRequest
*/
func (a *LocaltxsubmissionAPIService) LocaltxsubmissionTxPost(
	ctx context.Context,
) LocaltxsubmissionAPILocaltxsubmissionTxPostRequest {
	return LocaltxsubmissionAPILocaltxsubmissionTxPostRequest{
		ApiService: a,
		ctx:        ctx,
	}
}

// Execute executes the request
//
//	@return string
func (a *LocaltxsubmissionAPIService) LocaltxsubmissionTxPostExecute(
	r LocaltxsubmissionAPILocaltxsubmissionTxPostRequest,
) (string, *http.Response, error) {
	var (
		localVarHTTPMethod  = http.MethodPost
		localVarPostBody    interface{}
		formFiles           []formFile
		localVarReturnValue string
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(
		r.ctx,
		"LocaltxsubmissionAPIService.LocaltxsubmissionTxPost",
	)
	if err != nil {
		return localVarReturnValue, nil, &GenericOpenAPIError{
			error: err.Error(),
		}
	}

	localVarPath := localBasePath + "/localtxsubmission/tx"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := url.Values{}
	localVarFormParams := url.Values{}
	if r.contentType == nil {
		return localVarReturnValue, nil, reportError(
			"contentType is required and must be specified",
		)
	}

	// to determine the Content-Type header
	localVarHTTPContentTypes := []string{}

	// set Content-Type header
	localVarHTTPContentType := selectHeaderContentType(localVarHTTPContentTypes)
	if localVarHTTPContentType != "" {
		localVarHeaderParams["Content-Type"] = localVarHTTPContentType
	}

	// to determine the Accept header
	localVarHTTPHeaderAccepts := []string{"application/json"}

	// set Accept header
	localVarHTTPHeaderAccept := selectHeaderAccept(localVarHTTPHeaderAccepts)
	if localVarHTTPHeaderAccept != "" {
		localVarHeaderParams["Accept"] = localVarHTTPHeaderAccept
	}
	parameterAddToHeaderOrQuery(
		localVarHeaderParams,
		"Content-Type",
		r.contentType,
		"",
		"",
	)
	req, err := a.client.prepareRequest(
		r.ctx,
		localVarPath,
		localVarHTTPMethod,
		localVarPostBody,
		localVarHeaderParams,
		localVarQueryParams,
		localVarFormParams,
		formFiles,
	)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.client.callAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := io.ReadAll(localVarHTTPResponse.Body)
	localVarHTTPResponse.Body.Close()
	localVarHTTPResponse.Body = io.NopCloser(bytes.NewBuffer(localVarBody))
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: localVarHTTPResponse.Status,
		}
		if localVarHTTPResponse.StatusCode == 400 {
			var v string
			err = a.client.decode(
				&v,
				localVarBody,
				localVarHTTPResponse.Header.Get("Content-Type"),
			)
			if err != nil {
				newErr.error = err.Error()
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.error = formatErrorMessage(localVarHTTPResponse.Status, &v)
			newErr.model = v
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 415 {
			var v string
			err = a.client.decode(
				&v,
				localVarBody,
				localVarHTTPResponse.Header.Get("Content-Type"),
			)
			if err != nil {
				newErr.error = err.Error()
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.error = formatErrorMessage(localVarHTTPResponse.Status, &v)
			newErr.model = v
			return localVarReturnValue, localVarHTTPResponse, newErr
		}
		if localVarHTTPResponse.StatusCode == 500 {
			var v string
			err = a.client.decode(
				&v,
				localVarBody,
				localVarHTTPResponse.Header.Get("Content-Type"),
			)
			if err != nil {
				newErr.error = err.Error()
				return localVarReturnValue, localVarHTTPResponse, newErr
			}
			newErr.error = formatErrorMessage(localVarHTTPResponse.Status, &v)
			newErr.model = v
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.client.decode(
		&localVarReturnValue,
		localVarBody,
		localVarHTTPResponse.Header.Get("Content-Type"),
	)
	if err != nil {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}
