// Copyright 2026 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utxorpc

import (
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/alonzo"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common/script"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	"github.com/blinklabs-io/plutigo/data"
)

type upperOnlyTransaction struct {
	testTransaction
	era int
}

func (t upperOnlyTransaction) Type() int { return t.era }

func (upperOnlyTransaction) TTL() uint64 { return 100 }

func TestBuildTxInfoUpperBoundClosureByEra(t *testing.T) {
	var history localstatequery.EraHistoryResult
	history.Params.SlotLength = 1000
	for _, tc := range []struct {
		name   string
		era    int
		closed uint64
	}{
		{"Alonzo", alonzo.TxTypeAlonzo, 1},
		{"Babbage", babbage.TxTypeBabbage, 1},
		{"Conway", conway.TxTypeConway, 0},
	} {
		for _, version := range []struct {
			name  string
			value plutusScriptVersion
		}{{"V1", plutusScriptV1}, {"V2", plutusScriptV2}} {
			t.Run(tc.name+"/"+version.name, func(t *testing.T) {
				info, err := buildTxInfoForVersion(
					&upperOnlyTransaction{era: tc.era}, nil, 0,
					[]localstatequery.EraHistoryResult{history}, version.value,
				)
				if err != nil {
					t.Fatal(err)
				}
				var got data.PlutusData
				switch value := info.(type) {
				case script.TxInfoV1:
					got = value.ValidRange.ToPlutusData()
				case script.TxInfoV2:
					got = value.ValidRange.ToPlutusData()
				default:
					t.Fatalf("unexpected TxInfo type %T", info)
				}
				want := data.NewConstr(0,
					data.NewConstr(0, data.NewConstr(0), data.NewConstr(1)),
					data.NewConstr(0,
						data.NewConstr(1, data.NewInteger(big.NewInt(100_000))),
						data.NewConstr(tc.closed),
					),
				)
				if !got.Equal(want) {
					t.Fatalf("validity range = %s, want %s", got, want)
				}
			})
		}
	}
}
