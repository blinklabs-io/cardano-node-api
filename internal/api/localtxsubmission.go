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
	"fmt"
	"io"

	"github.com/blinklabs-io/gouroboros/protocol/localtxsubmission"
	"github.com/blinklabs-io/tx-submit-api/submit"
	"github.com/gin-gonic/gin"

	"github.com/blinklabs-io/cardano-node-api/internal/config"
	"github.com/blinklabs-io/cardano-node-api/internal/logging"
)

func configureLocalTxSubmissionRoutes(apiGroup *gin.RouterGroup) {
	group := apiGroup.Group("/localtxsubmission")
	group.POST("/tx", handleLocalSubmitTx)
}

// handleLocalSubmitTx godoc
//
//	@Summary		Submit Tx
//	@Tags			localtxsubmission
//	@Description	Submit an already serialized transaction to the network.
//	@Produce		json
//	@Param			Content-Type	header		string	true	"Content type"	Enums(application/cbor)
//	@Success		202				{object}	string	"Ok"
//	@Failure		400				{object}	string	"Bad Request"
//	@Failure		415				{object}	string	"Unsupported Media Type"
//	@Failure		500				{object}	string	"Server Error"
//	@Router			/localtxsubmission/tx [post]
func handleLocalSubmitTx(c *gin.Context) {
	// First, initialize our configuration and loggers
	cfg := config.GetConfig()
	logger := logging.GetLogger()
	// Check our headers for content-type
	if c.ContentType() != "application/cbor" {
		// Log the error, return an error to the user, and increment failed count
		logger.Error("invalid request body, should be application/cbor")
		c.JSON(415, "invalid request body, should be application/cbor")
		// _ = ginmetrics.GetMonitor().GetMetric("tx_submit_fail_count").Inc(nil)
		return
	}
	// Read raw transaction bytes from the request body and store in a byte array
	txRawBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		// Log the error, return an error to the user, and increment failed count
		logger.Error("failed to read request body:", "error", err)
		c.JSON(500, "failed to read request body")
		// _ = ginmetrics.GetMonitor().GetMetric("tx_submit_fail_count").Inc(nil)
		return
	}
	// Close request body after read
	if c.Request.Body != nil {
		if err = c.Request.Body.Close(); err != nil {
			logger.Error("failed to close request body:", "error", err)
		}
	}
	// Send TX
	errorChan := make(chan error)
	submitConfig := &submit.Config{
		ErrorChan:    errorChan,
		NetworkMagic: cfg.Node.NetworkMagic,
		NodeAddress:  cfg.Node.Address,
		NodePort:     cfg.Node.Port,
		SocketPath:   cfg.Node.SocketPath,
		Timeout:      cfg.Node.Timeout,
	}
	txHash, err := submit.SubmitTx(submitConfig, txRawBytes)
	if err != nil {
		if c.GetHeader("Accept") == "application/cbor" {
			txRejectErr := err.(localtxsubmission.TransactionRejectedError)
			c.Data(400, "application/cbor", txRejectErr.ReasonCbor)
		} else {
			if err.Error() != "" {
				c.JSON(400, err.Error())
			} else {
				c.JSON(400, fmt.Sprintf("%s", err))
			}
		}
		// _ = ginmetrics.GetMonitor().GetMetric("tx_submit_fail_count").Inc(nil)
		return
	}
	// Start async error handler
	go func() {
		err, ok := <-errorChan
		if ok {
			logger.Error("failure communicating with node:", "error", err)
			c.JSON(500, "failure communicating with node")
			// _ = ginmetrics.GetMonitor().GetMetric("tx_submit_fail_count").Inc(nil)
		}
	}()
	// Return transaction ID
	c.JSON(202, txHash)
	// Increment custom metric
	// _ = ginmetrics.GetMonitor().GetMetric("tx_submit_count").Inc(nil)
}
