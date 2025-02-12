/*
cardano-node-api

Testing ChainsyncAPIService

*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech);

package openapi

import (
	"context"
	openapiclient "github.com/blinklabs-io/cardano-node-api/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_openapi_ChainsyncAPIService(t *testing.T) {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)

	t.Run("Test ChainsyncAPIService ChainsyncSyncGet", func(t *testing.T) {

		t.Skip("skip test") // remove to run test

		httpRes, err := apiClient.ChainsyncAPI.ChainsyncSyncGet(context.Background()).
			Execute()

		require.Nil(t, err)
		assert.Equal(t, 200, httpRes.StatusCode)

	})

}
