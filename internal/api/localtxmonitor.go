package api

import (
	"encoding/hex"
	"net/http"

	"github.com/blinklabs-io/cardano-node-api/internal/node"

	"github.com/fxamacker/cbor/v2"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/blake2b"
)

func configureLocalTxMonitorRoutes(apiGroup *gin.RouterGroup) {
	group := apiGroup.Group("/localtxmonitor")
	group.GET("/sizes", handleLocalTxMonitorSizes)
	group.GET("/has_tx/:tx_hash", handleLocalTxMonitorHasTx)
	group.GET("/txs", handleLocalTxMonitorTxs)
}

type responseLocalTxMonitorSizes struct {
	Capacity uint32 `json:"capacity"`
	Size     uint32 `json:"size"`
	TxCount  uint32 `json:"tx_count"`
}

// handleLocalTxMonitorSizes godoc
// @Summary      Get mempool capacity, size, and TX count
// @Tags         localtxmonitor
// @Accept       json
// @Produce      json
// @Success      200  {object}  responseLocalTxMonitorSizes
// @Failure      500  {object}  responseApiError
// @Router       /localtxmonitor/sizes [get]
func handleLocalTxMonitorSizes(c *gin.Context) {
	// Connect to node
	oConn, err := node.GetConnection()
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
	oConn.LocalTxMonitor().Client.Start()
	// Get sizes
	capacity, size, txCount, err := oConn.LocalTxMonitor().Client.GetSizes()
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}
	// Create response
	resp := responseLocalTxMonitorSizes{
		Capacity: capacity,
		Size:     size,
		TxCount:  txCount,
	}
	c.JSON(200, resp)
}

type requestLocalTxMonitorHasTx struct {
	TxHash string `uri:"tx_hash" binding:"required"`
}

type responseLocalTxMonitorHasTx struct {
	HasTx bool `json:"has_tx"`
}

// handleLocalTxMonitorHasTx godoc
// @Summary      Check if a particular TX exists in the mempool
// @Tags         localtxmonitor
// @Accept       json
// @Produce      json
// @Success      200  {object}  responseLocalTxMonitorHasTx
// @Failure      500  {object}  responseApiError
// @Router       /localtxmonitor/has_tx/{tx_hash} [get]
func handleLocalTxMonitorHasTx(c *gin.Context) {
	// Get parameters
	var req requestLocalTxMonitorHasTx
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, apiError(err.Error()))
		return
	}
	// Connect to node
	oConn, err := node.GetConnection()
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
	oConn.LocalTxMonitor().Client.Start()
	// Make the call to the node
	txHash, err := hex.DecodeString(req.TxHash)
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}
	hasTx, err := oConn.LocalTxMonitor().Client.HasTx(txHash)
	if err != nil {
		c.JSON(500, apiError(err.Error()))
		return
	}
	// Create response
	resp := responseLocalTxMonitorHasTx{
		HasTx: hasTx,
	}
	c.JSON(200, resp)
}

type responseLocalTxMonitorTxs struct {
	TxHash  string `json:"tx_hash" swaggertype:"string" format:"base16" example:"96649a8b827a5a4d508cd4e98cd88832482f7b884d507a49466d1fb8c4b14978"`
	TxBytes []byte `json:"tx_bytes" swaggertype:"string" format:"base64" example:"<base64 encoded transaction bytes>"`
}

// handleLocalTxMonitorTxs godoc
// @Summary      List all transactions in the mempool
// @Tags         localtxmonitor
// @Accept       json
// @Produce      json
// @Success      200  {object}  []responseLocalTxMonitorTxs
// @Failure      500  {object}  responseApiError
// @Router       /localtxmonitor/txs [get]
func handleLocalTxMonitorTxs(c *gin.Context) {
	// Connect to node
	oConn, err := node.GetConnection()
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
	oConn.LocalTxMonitor().Client.Start()
	// Collect TX hashes
	resp := []responseLocalTxMonitorTxs{}
	for {
		tx, err := oConn.LocalTxMonitor().Client.NextTx()
		if err != nil {
			c.JSON(500, apiError(err.Error()))
			return
		}
		if tx == nil {
			break
		}
		// Unwrap raw transaction bytes into a CBOR array
		var txUnwrap []cbor.RawMessage
		if err := cbor.Unmarshal(tx, &txUnwrap); err != nil {
			c.JSON(500, apiError(err.Error()))
			return
		}
		// index 0 is the transaction body
		txBody := txUnwrap[0]
		// Hash the TX body with blake2b256 to get TX hash
		txHash := blake2b.Sum256(txBody)
		// Encode TX hash as hex
		txHashHex := hex.EncodeToString(txHash[:])
		// Add to response
		resp = append(
			resp,
			responseLocalTxMonitorTxs{
				TxHash:  txHashHex,
				TxBytes: tx,
			},
		)
	}
	// Send response
	c.JSON(200, resp)
}
