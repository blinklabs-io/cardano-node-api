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

package utxorpc

import (
	"encoding/hex"

	"github.com/blinklabs-io/gouroboros/ledger"
	cardano "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
)

func NewBlockHeaderFromBlock(b ledger.Block) cardano.BlockHeader {
	var header cardano.BlockHeader
	header.Slot = b.SlotNumber()
	tmpHash, _ := hex.DecodeString(b.Hash())
	header.Hash = tmpHash
	header.Height = b.BlockNumber()
	return header
}

func NewBlockBodyFromBlock(b ledger.Block) cardano.BlockBody {
	var body cardano.BlockBody
	var txs []*cardano.Tx
	for _, t := range b.Transactions() {
		tx := NewTxFromTransaction(t)
		txs = append(txs, &tx)
	}
	body.Tx = txs
	return body
}

func NewBlockFromBlock(b ledger.Block) cardano.Block {
	var block cardano.Block
	body := NewBlockBodyFromBlock(b)
	header := NewBlockHeaderFromBlock(b)
	block.Body = &body
	block.Header = &header
	return block
}

func NewTxFromTransaction(t ledger.Transaction) cardano.Tx {
	var tx cardano.Tx

	var txi []*cardano.TxInput
	for _, i := range t.Inputs() {
		input := NewTxInputFromTransactionInput(i)
		txi = append(txi, &input)
	}
	tx.Inputs = txi

	var txo []*cardano.TxOutput
	for _, o := range t.Outputs() {
		output := NewTxOutputFromTransactionOutput(o)
		txo = append(txo, &output)
	}
	tx.Outputs = txo

	// tx.Certificates = t.Certificates()
	// tx.Withdrawals = t.Withdrawals()
	// tx.Mint = t.Mint()
	// tx.ReferenceInputs = t.ReferenceInputs()
	// tx.Witnesses = t.Witnesses()
	// tx.Collateral = t.Collateral()
	tx.Fee = t.Fee()
	// tx.Validity = t.Validity()
	// tx.Successful = t.Successful()
	// tx.Auxiliary = t.AuxData()
	tmpHash, _ := hex.DecodeString(t.Hash())
	tx.Hash = tmpHash
	return tx
}

// cardano.TxValidity
// type TxValidity struct {
// 	Start uint64
// 	Ttl   uint64
// }

func NewTxInputFromTransactionInput(i ledger.TransactionInput) cardano.TxInput {
	var txInput cardano.TxInput
	txInput.TxHash = i.Id().Bytes()
	txInput.OutputIndex = i.Index()
	// txInput.AsOutput = i.AsOutput()
	// txInput.Redeemer = i.Redeemer()
	return txInput
}

// cardano.Redeemer
// type Redeemer struct {
// 	Purpose RedeemerPurpose
// 	Datum   *PlutusData
// }

func NewTxOutputFromTransactionOutput(o ledger.TransactionOutput) cardano.TxOutput {
	var txOut cardano.TxOutput
	txOut.Address = o.Address().Bytes()
	txOut.Coin = o.Amount()
	// txOut.Assets = o.Assets()
	// txOut.Datum = o.Datum()
	// txOut.DatumHash = o.DatumHash()
	// txOut.Script = o.Script()
	return txOut
}

// func NewMultiassetListFromMultiAsset(m ledger.MultiAsset[T]) []cardano.Multiasset {
// 	var assets []cardano.Multiasset
// 	for _, policyId := range m.Policies() {
// 		var multiAsset cardano.Multiasset
// 		multiAsset.PolicyId = policyId
// 		for _, assetName := range m.Assets(policyId) {
// 			var asset cardano.Asset
// 			asset.Name = assetName
// 			v := m.Asset(policyId, assetName)
// 			switch v.(type) {
// 			case int64:
// 				asset.MintCoin = v
// 			case uint64:
// 				asset.OutputCoin = v
// 			}
// 			multiAsset.Assets = append(multiAsset.Assets, asset)
// 		}
// 		assets = append(assets, multiAsset)
// 	}
// 	return assets
// }
