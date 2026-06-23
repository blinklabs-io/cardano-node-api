// Copyright 2026 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	"github.com/gin-gonic/gin"
)

const testPolicyIDHex = "00000000000000000000000000000000000000000000000000000000"

func TestSearchUTxOsByAssetRequiresAssetNamePresence(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/localstatequery/utxos/search-by-asset?policy_id="+testPolicyIDHex,
		nil,
	)

	searchUTxOsByAsset(c, 0)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "asset_name parameter is required") {
		t.Fatalf("expected missing asset_name error, got %s", w.Body.String())
	}
}

func TestSearchUTxOsByAssetAllowsEmptyAssetName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/localstatequery/utxos/search-by-asset?policy_id="+testPolicyIDHex+"&asset_name=",
		nil,
	)

	// maxResults <= 0 requires the address parameter, which lets the
	// validation path complete without a node connection
	searchUTxOsByAsset(c, 0)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "address parameter is required") {
		t.Fatalf("expected address requirement error, got %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "asset_name parameter is required") {
		t.Fatalf("empty asset_name was treated as missing: %s", w.Body.String())
	}
}

func TestFilterUTxOsByAssetResultsDoesNotTruncateAtExactLimit(t *testing.T) {
	var policyId ledger.Blake2b224
	assetName := []byte("asset")
	utxos := testUTxOsWithAsset(t, 2, policyId, assetName)

	results, truncated := filterUTxOsByAssetResults(
		utxos,
		policyId,
		assetName,
		true,
		2,
	)

	if truncated {
		t.Fatal("expected exact-limit result not to be truncated")
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
}

func TestFilterUTxOsByAssetResultsTruncatesOnlyAfterExtraMatch(t *testing.T) {
	var policyId ledger.Blake2b224
	assetName := []byte("asset")
	utxos := testUTxOsWithAsset(t, 3, policyId, assetName)

	results, truncated := filterUTxOsByAssetResults(
		utxos,
		policyId,
		assetName,
		true,
		2,
	)

	if !truncated {
		t.Fatal("expected result to be truncated after an extra matching UTxO")
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
}

func testUTxOsWithAsset(
	t *testing.T,
	count int,
	policyId ledger.Blake2b224,
	assetName []byte,
) map[localstatequery.UtxoId]ledger.BabbageTransactionOutput {
	t.Helper()

	ret := make(map[localstatequery.UtxoId]ledger.BabbageTransactionOutput)
	for i := range count {
		var txHash ledger.Blake2b256
		txHash[31] = byte(i + 1)
		ret[localstatequery.UtxoId{
			Hash: txHash,
			Idx:  i,
		}] = testOutputWithAsset(t, policyId, assetName)
	}
	return ret
}

func testOutputWithAsset(
	t *testing.T,
	policyId ledger.Blake2b224,
	assetName []byte,
) ledger.BabbageTransactionOutput {
	t.Helper()

	paymentHash := make([]byte, lcommon.AddressHashSize)
	addr, err := ledger.NewAddressFromParts(
		ledger.AddressTypeKeyNone,
		lcommon.AddressNetworkTestnet,
		paymentHash,
		nil,
	)
	if err != nil {
		t.Fatalf("NewAddressFromParts() error = %v", err)
	}
	assets := lcommon.NewMultiAsset[lcommon.MultiAssetTypeOutput](
		map[lcommon.Blake2b224]map[cbor.ByteString]lcommon.MultiAssetTypeOutput{
			policyId: {
				cbor.NewByteString(assetName): big.NewInt(1),
			},
		},
	)
	return ledger.BabbageTransactionOutput{
		OutputAddress: addr,
		OutputAmount: ledger.MaryTransactionOutputValue{
			Amount: 1000000,
			Assets: &assets,
		},
	}
}
