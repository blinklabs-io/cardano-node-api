// Copyright 2024 Blink Labs Software
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

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/gin-gonic/gin"

	"github.com/blinklabs-io/cardano-node-api/internal/node"
)

func configureLocalStateQueryRoutes(apiGroup *gin.RouterGroup) {
	group := apiGroup.Group("/localstatequery")
	group.GET("/current-era", handleLocalStateQueryCurrentEra)
	group.GET("/system-start", handleLocalStateQuerySystemStart)
	group.GET("/tip", handleLocalStateQueryTip)
	group.GET("/era-history", handleLocalStateQueryEraHistory)
	group.GET("/protocol-params", handleLocalStateQueryProtocolParams)
	// TODO: uncomment after this is fixed:
	// - https://github.com/blinklabs-io/gouroboros/issues/584
	// group.GET("/genesis-config", handleLocalStateQueryGenesisConfig)
}

type responseLocalStateQueryCurrentEra struct {
	Id   uint8  `json:"id"`
	Name string `json:"name"`
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

	// Create response
	era := ledger.GetEraById(uint8(eraNum))
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

	// Create response
	resp := responseLocalStateQuerySystemStart{
		Year:        result.Year,
		Day:         result.Day,
		Picoseconds: result.Picoseconds,
	}
	c.JSON(200, resp)
}

type responseLocalStateQueryTip struct {
	Era     string `json:"era"`
	EpochNo int    `json:"epoch_no"`
	BlockNo int64  `json:"block_no"`
	Slot    uint64 `json:"slot_no"`
	Hash    string `json:"hash"`
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
type responseLocalStateQueryEraHistory struct {
}

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
type responseLocalStateQueryProtocolParams struct {
}

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
type responseLocalStateQueryGenesisConfig struct {
}

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
