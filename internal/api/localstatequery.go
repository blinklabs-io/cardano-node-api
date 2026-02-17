// Copyright 2025 Blink Labs Software
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
	"encoding/hex"
	"math"

	"github.com/blinklabs-io/cardano-node-api/internal/node"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	"github.com/gin-gonic/gin"
)

func configureLocalStateQueryRoutes(apiGroup *gin.RouterGroup) {
	group := apiGroup.Group("/localstatequery")
	group.GET("/current-era", handleLocalStateQueryCurrentEra)
	group.GET("/system-start", handleLocalStateQuerySystemStart)
	group.GET("/tip", handleLocalStateQueryTip)
	group.GET("/era-history", handleLocalStateQueryEraHistory)
	group.GET("/protocol-params", handleLocalStateQueryProtocolParams)
	group.GET("/utxos/search-by-asset", handleLocalStateQuerySearchUTxOsByAsset)
	// TODO: uncomment after this is fixed:
	// - https://github.com/blinklabs-io/gouroboros/issues/584
	// group.GET("/genesis-config", handleLocalStateQueryGenesisConfig)
}

type responseLocalStateQueryCurrentEra struct {
	Name string `json:"name"`
	Id   uint8  `json:"id"`
}

// handleLocalStateQueryCurrentEra godoc
//
//	@Summary	Query Current Era
//	@Tags		localstatequery
//	@Produce	json
//	@Success	200	{object}	responseLocalStateQueryCurrentEra
//	@Failure	500	{object}	responseApiError
//	@Router		/localstatequery/current-era [get]
func handleLocalStateQueryCurrentEra(c *gin.Context) {
	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}
	// Async error handler
	go func() {
		err, ok := <-oConn.ErrorChan()
		if !ok {
			return
		}
		c.JSON(500, apiError(err.Error()))
	}()
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()
	// Start client
	oConn.LocalStateQuery().Client.Start()

	// Get era
	eraNum, err := oConn.LocalStateQuery().Client.GetCurrentEra()
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}

	if eraNum < 0 || eraNum > math.MaxUint8 {
		c.JSON(500, apiError("era number int overflow"))
		return
	}
	era := ledger.GetEraById(uint8(eraNum))

	// Create response
	resp := responseLocalStateQueryCurrentEra{
		Id:   era.Id,
		Name: era.Name,
	}
	c.JSON(200, resp)
}

type responseLocalStateQuerySystemStart struct {
	Year        int    `json:"year"`
	Day         int    `json:"day"`
	Picoseconds uint64 `json:"picoseconds"`
}

// handleLocalStateQuerySystemStart godoc
//
//	@Summary	Query System Start
//	@Tags		localstatequery
//	@Produce	json
//	@Success	200	{object}	responseLocalStateQuerySystemStart
//	@Failure	500	{object}	responseApiError
//	@Router		/localstatequery/system-start [get]
func handleLocalStateQuerySystemStart(c *gin.Context) {
	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}
	// Async error handler
	go func() {
		err, ok := <-oConn.ErrorChan()
		if !ok {
			return
		}
		c.JSON(500, apiError(err.Error()))
	}()
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()
	// Start client
	oConn.LocalStateQuery().Client.Start()

	// Get system start
	result, err := oConn.LocalStateQuery().Client.GetSystemStart()
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}
	// Validate data before conversion
	if result.Year.Int64() > math.MaxInt {
		c.JSON(500, apiError("invalid date conversion"))
		return
	}
	// Create response
	resp := responseLocalStateQuerySystemStart{
		Year:        int(result.Year.Int64()),
		Day:         result.Day,
		Picoseconds: result.Picoseconds.Uint64(),
	}
	c.JSON(200, resp)
}

type responseLocalStateQueryTip struct {
	Era     string `json:"era"`
	Hash    string `json:"hash"`
	EpochNo int    `json:"epoch_no"`
	BlockNo int64  `json:"block_no"`
	Slot    uint64 `json:"slot_no"`
}

// handleLocalStateQueryTip godoc
//
//	@Summary	Query Chain Tip
//	@Tags		localstatequery
//	@Produce	json
//	@Success	200	{object}	responseLocalStateQueryTip
//	@Failure	500	{object}	responseApiError
//	@Router		/localstatequery/tip [get]
func handleLocalStateQueryTip(c *gin.Context) {
	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}
	// Async error handler
	go func() {
		err, ok := <-oConn.ErrorChan()
		if !ok {
			return
		}
		c.JSON(500, apiError(err.Error()))
	}()
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()
	// Start client
	oConn.LocalStateQuery().Client.Start()

	// Get era
	eraNum, err := oConn.LocalStateQuery().Client.GetCurrentEra()
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}
	if eraNum < 0 || eraNum > math.MaxUint8 {
		c.JSON(500, apiError("era number int overflow"))
		return
	}
	era := ledger.GetEraById(uint8(eraNum))

	// Get epochNo
	epochNo, err := oConn.LocalStateQuery().Client.GetEpochNo()
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}

	// Get blockNo
	blockNo, err := oConn.LocalStateQuery().Client.GetChainBlockNo()
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}

	// Get chain point (slot and hash)
	point, err := oConn.LocalStateQuery().Client.GetChainPoint()
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}

	// Create response
	resp := responseLocalStateQueryTip{
		Era:     era.Name,
		EpochNo: epochNo,
		BlockNo: blockNo,
		Slot:    point.Slot,
		Hash:    hex.EncodeToString(point.Hash),
	}
	c.JSON(200, resp)
}

// TODO: fill this in
//
//nolint:unused
type responseLocalStateQueryEraHistory struct{}

// handleLocalStateQueryEraHistory godoc
//
//	@Summary	Query Era History
//	@Tags		localstatequery
//	@Produce	json
//	@Success	200	{object}	responseLocalStateQueryEraHistory
//	@Failure	500	{object}	responseApiError
//	@Router		/localstatequery/era-history [get]
func handleLocalStateQueryEraHistory(c *gin.Context) {
	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}
	// Async error handler
	go func() {
		err, ok := <-oConn.ErrorChan()
		if !ok {
			return
		}
		c.JSON(500, apiError(err.Error()))
	}()
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()
	// Start client
	oConn.LocalStateQuery().Client.Start()

	// Get eraHistory
	eraHistory, err := oConn.LocalStateQuery().Client.GetEraHistory()
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}

	// Create response
	//resp := responseLocalStateQueryProtocolParams{
	//}
	c.JSON(200, eraHistory)
}

// TODO: fill this in
//
//nolint:unused
type responseLocalStateQueryProtocolParams struct{}

// handleLocalStateQueryProtocolParams godoc
//
//	@Summary	Query Current Protocol Parameters
//	@Tags		localstatequery
//	@Produce	json
//	@Success	200	{object}	responseLocalStateQueryProtocolParams
//	@Failure	500	{object}	responseApiError
//	@Router		/localstatequery/protocol-params [get]
func handleLocalStateQueryProtocolParams(c *gin.Context) {
	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}
	// Async error handler
	go func() {
		err, ok := <-oConn.ErrorChan()
		if !ok {
			return
		}
		c.JSON(500, apiError(err.Error()))
	}()
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()
	// Start client
	oConn.LocalStateQuery().Client.Start()

	// Get protoParams
	protoParams, err := oConn.LocalStateQuery().Client.GetCurrentProtocolParams()
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}

	// Create response
	//resp := responseLocalStateQueryProtocolParams{
	//}
	c.JSON(200, protoParams)
}

// TODO: fill this in
//
//nolint:unused
type responseLocalStateQueryGenesisConfig struct{}

// handleLocalStateQueryGenesisConfig godoc
//
//	@Summary	Query Genesis Config
//	@Tags		localstatequery
//	@Produce	json
//	@Success	200	{object}	responseLocalStateQueryGenesisConfig
//	@Failure	500	{object}	responseApiError
//	@Router		/localstatequery/genesis-config [get]
//
//nolint:unused
func handleLocalStateQueryGenesisConfig(c *gin.Context) {
	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}
	// Async error handler
	go func() {
		err, ok := <-oConn.ErrorChan()
		if !ok {
			return
		}
		c.JSON(500, apiError(err.Error()))
	}()
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()
	// Start client
	oConn.LocalStateQuery().Client.Start()

	// Get genesisConfig
	genesisConfig, err := oConn.LocalStateQuery().Client.GetGenesisConfig()
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}

	// Create response
	//resp := responseLocalStateQueryGenesisConfig{
	//}
	c.JSON(200, genesisConfig)
}

type responseLocalStateQuerySearchUTxOsByAsset struct {
	UTxOs []utxoItem `json:"utxos"`
	Count int        `json:"count"`
}

type utxoItem struct {
	TxHash  string `json:"tx_hash"`
	Index   uint32 `json:"index"`
	Address string `json:"address"`
	Amount  uint64 `json:"amount"           format:"int64" example:"1000000"`
	Assets  any    `json:"assets,omitempty"`
}

// handleLocalStateQuerySearchUTxOsByAsset godoc
//
//	@Summary	Search UTxOs by Asset
//	@Tags		localstatequery
//	@Produce	json
//	@Param		policy_id	query		string	true	"Policy ID (hex)"
//	@Param		asset_name	query		string	true	"Asset name (hex)"
//	@Param		address		query		string	false	"Optional: Filter by address"
//	@Success	200			{object}	responseLocalStateQuerySearchUTxOsByAsset
//	@Failure	400			{object}	responseApiError
//	@Failure	500			{object}	responseApiError
//	@Router		/localstatequery/utxos/search-by-asset [get]
func handleLocalStateQuerySearchUTxOsByAsset(c *gin.Context) {
	// Get query parameters
	policyIdHex := c.Query("policy_id")
	assetNameHex := c.Query("asset_name")
	addressStr := c.Query("address")

	// Validate required parameters
	if policyIdHex == "" {
		c.JSON(400, apiError("policy_id parameter is required"))
		return
	}
	if assetNameHex == "" {
		c.JSON(400, apiError("asset_name parameter is required"))
		return
	}

	// Parse policy ID (28 bytes)
	policyIdBytes, err := hex.DecodeString(policyIdHex)
	if err != nil {
		c.JSON(400, apiError("invalid policy_id hex: "+err.Error()))
		return
	}
	if len(policyIdBytes) != 28 {
		c.JSON(400, apiError("policy_id must be 28 bytes"))
		return
	}
	var policyId ledger.Blake2b224
	copy(policyId[:], policyIdBytes)

	// Parse asset name
	assetName, err := hex.DecodeString(assetNameHex)
	if err != nil {
		c.JSON(400, apiError("invalid asset_name hex: "+err.Error()))
		return
	}

	// Parse optional address
	var addrs []ledger.Address
	if addressStr != "" {
		addr, err := ledger.NewAddress(addressStr)
		if err != nil {
			c.JSON(400, apiError("invalid address: "+err.Error()))
			return
		}
		addrs = append(addrs, addr)
	}

	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}
	// Async error handler
	go func() {
		err, ok := <-oConn.ErrorChan()
		if !ok {
			return
		}
		c.JSON(500, apiError(err.Error()))
	}()
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()
	// Start client
	oConn.LocalStateQuery().Client.Start()

	// Get UTxOs (either by address or whole set)
	var utxos *localstatequery.UTxOsResult
	if len(addrs) > 0 {
		utxos, err = oConn.LocalStateQuery().Client.GetUTxOByAddress(addrs)
	} else {
		utxos, err = oConn.LocalStateQuery().Client.GetUTxOWhole()
	}
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}

	// Filter UTxOs by asset
	results := make([]utxoItem, 0)
	for utxoId, output := range utxos.Results {
		// Check if output has assets
		assets := output.Assets()
		if assets == nil {
			continue
		}

		// Check if the asset exists in this UTxO
		amount := assets.Asset(policyId, assetName)
		if amount != nil && amount.Sign() > 0 {
			item := utxoItem{
				TxHash:  hex.EncodeToString(utxoId.Hash[:]),
				Index:   uint32(utxoId.Idx), // #nosec G115
				Address: output.Address().String(),
				Amount:  output.Amount().Uint64(),
				Assets:  assets,
			}
			results = append(results, item)
		}
	}

	// Create response
	resp := responseLocalStateQuerySearchUTxOsByAsset{
		UTxOs: results,
		Count: len(results),
	}
	c.JSON(200, resp)
}
