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
	"net/http"

	"github.com/blinklabs-io/cardano-node-api/internal/node"

	"github.com/blinklabs-io/adder/event"
	ocommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func configureChainSyncRoutes(apiGroup *gin.RouterGroup) {
	group := apiGroup.Group("/chainsync")
	group.GET("/sync", handleChainSyncSync)
}

type requestChainSyncSync struct {
	Slot uint64 `form:"slot"`
	Hash string `form:"hash"`
	Tip  bool   `form:"tip"`
}

// handleChainSyncSync godoc
//
//	@Summary	Start a chain-sync using a websocket for events
//	@Tags		chainsync
//	@Success	101
//	@Failure	400		{object}	responseApiError
//	@Failure	500		{object}	responseApiError
//	@Param		tip		query		bool	false	"whether to start from the current tip"
//	@Param		slot	query		int		false	"slot to start sync at, should match hash"
//	@Param		hash	query		string	false	"block hash to start sync at, should match slot"
//	@Router		/chainsync/sync [get]
func handleChainSyncSync(c *gin.Context) {
	// Get parameters
	var req requestChainSyncSync
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}
	if !req.Tip && (req.Slot == 0 || req.Hash == "") {
		c.JSON(http.StatusBadRequest, apiError("you must provide the 'slot' and 'hash' parameters or set 'tip' to True"))
		return
	}
	// Setup event channel
	eventChan := make(chan event.Event, 10)
	connCfg := node.ConnectionConfig{
		ChainSyncEventChan: eventChan,
	}
	// Connect to node
	oConn, err := node.GetConnection(&connCfg)
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
	var intersectPoints []ocommon.Point
	if req.Tip {
		tip, err := oConn.ChainSync().Client.GetCurrentTip()
		if err != nil {
			c.JSON(500, apiError(err.Error()))
			return
		}
		intersectPoints = []ocommon.Point{
			tip.Point,
		}
	} else {
		hashBytes, err := hex.DecodeString(req.Hash)
		if err != nil {
			c.JSON(500, apiError(err.Error()))
			return
		}
		intersectPoints = []ocommon.Point{
			ocommon.NewPoint(req.Slot, hashBytes),
		}
	}
	// Start the sync with the node
	if err := oConn.ChainSync().Client.Sync(intersectPoints); err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}
	// Upgrade the connection
	webConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer webConn.Close()
	// Wait for events
	for {
		evt, ok := <-eventChan
		if !ok {
			return
		}
		if err := webConn.WriteJSON(evt); err != nil {
			c.JSON(500, apiError(err.Error()))
			return
		}
	}
}
