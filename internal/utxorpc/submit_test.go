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

package utxorpc

import (
	"bytes"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	script "github.com/blinklabs-io/gouroboros/ledger/common/script"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	"github.com/blinklabs-io/plutigo/data"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
)

func TestBuildScriptPurposeUnsupportedTags(t *testing.T) {
	testCases := []struct {
		name    string
		tag     lcommon.RedeemerTag
		wantErr string
	}{
		{
			name:    "cert redeemer returns error instead of dummy purpose",
			tag:     lcommon.RedeemerTagCert,
			wantErr: "not supported yet",
		},
		{
			name:    "reward redeemer returns error instead of dummy purpose",
			tag:     lcommon.RedeemerTagReward,
			wantErr: "not supported yet",
		},
		{
			name:    "unknown redeemer tag returns error",
			tag:     lcommon.RedeemerTagVoting,
			wantErr: "not supported yet",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			purpose, err := buildScriptPurposeForContext(
				&testTransaction{},
				map[string]ledger.Utxo{},
				tc.tag,
				0,
				nil,
			)
			if err == nil {
				t.Fatalf(
					"expected error for redeemer tag %d, got purpose %v",
					tc.tag,
					purpose,
				)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf(
					"expected error containing %q, got %q",
					tc.wantErr,
					err.Error(),
				)
			}
		})
	}
}

func TestBuildTxInfoErrorsOnUnresolvedInput(t *testing.T) {
	input := testInput("01", 0)
	tx := &testTransaction{
		inputs: []lcommon.TransactionInput{input},
	}

	_, err := buildTxInfoForVersion(
		tx,
		map[string]ledger.Utxo{},
		0,
		nil,
		0,
		plutusScriptV2,
	)
	if err == nil {
		t.Fatal("expected unresolved input error")
	}
	if !strings.Contains(err.Error(), "unresolved input") {
		t.Fatalf("expected unresolved input error, got %q", err)
	}
	if !strings.Contains(err.Error(), input.String()) {
		t.Fatalf("expected error to include input %q, got %q", input, err)
	}
}

func TestBuildTxInfoErrorsOnUnresolvedReferenceInput(t *testing.T) {
	referenceInput := testInput("02", 0)
	tx := &testTransaction{
		referenceInputs: []lcommon.TransactionInput{referenceInput},
	}

	_, err := buildTxInfoForVersion(
		tx,
		map[string]ledger.Utxo{},
		0,
		nil,
		0,
		plutusScriptV2,
	)
	if err == nil {
		t.Fatal("expected unresolved reference input error")
	}
	if !strings.Contains(err.Error(), "unresolved input") {
		t.Fatalf("expected unresolved input error, got %q", err)
	}
	if !strings.Contains(err.Error(), referenceInput.String()) {
		t.Fatalf(
			"expected error to include reference input %q, got %q",
			referenceInput,
			err,
		)
	}
}

func TestConvertPlutusDataNegativeBignumEncoding(t *testing.T) {
	negPastInt64 := new(big.Int).Sub(
		big.NewInt(math.MinInt64),
		big.NewInt(1),
	)
	negTwo64 := new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 64))

	testCases := []struct {
		name string
		in   *big.Int
		want []byte
	}{
		{
			name: "below int64 min stores minus-one complement",
			in:   negPastInt64,
			want: []byte{0x80, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "negative two64 stores cbor tag3 payload",
			in:   negTwo64,
			want: []byte{
				0xff,
				0xff,
				0xff,
				0xff,
				0xff,
				0xff,
				0xff,
				0xff,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := convertPlutusData(data.NewInteger(tc.in))
			if err != nil {
				t.Fatalf("convertPlutusData() error = %v", err)
			}
			gotBigInt := got.GetBigInt()
			if gotBigInt == nil {
				t.Fatalf("got %T, want PlutusData_BigInt", got.GetPlutusData())
			}
			bigN, ok := gotBigInt.GetBigInt().(*cardano.BigInt_BigNInt)
			if !ok {
				t.Fatalf(
					"got %T, want *cardano.BigInt_BigNInt",
					gotBigInt.GetBigInt(),
				)
			}
			if !bytes.Equal(bigN.BigNInt, tc.want) {
				t.Fatalf("BigNInt = %x, want %x", bigN.BigNInt, tc.want)
			}
		})
	}
}

func TestConvertPlutusDataMinInt64UsesInt(t *testing.T) {
	got, err := convertPlutusData(data.NewInteger(big.NewInt(math.MinInt64)))
	if err != nil {
		t.Fatalf("convertPlutusData() error = %v", err)
	}
	gotBigInt := got.GetBigInt()
	if gotBigInt == nil {
		t.Fatalf("got %T, want PlutusData_BigInt", got.GetPlutusData())
	}
	gotInt, ok := gotBigInt.GetBigInt().(*cardano.BigInt_Int)
	if !ok {
		t.Fatalf(
			"got %T, want *cardano.BigInt_Int",
			gotBigInt.GetBigInt(),
		)
	}
	if gotInt.Int != math.MinInt64 {
		t.Fatalf("Int = %d, want %d", gotInt.Int, int64(math.MinInt64))
	}
}

// TestSlotToPOSIXTimeUsesMillisecondSlotLength guards against double-scaling
// the slot length. The hard-fork era history query already reports SlotLength
// in milliseconds, so slotToPOSIXTime must not multiply it by 1000 again.
func TestSlotToPOSIXTimeUsesMillisecondSlotLength(t *testing.T) {
	const systemStartMs = int64(1_000_000)
	const slotLengthMs = 1000 // Shelley: 1 second per slot, reported as 1000 ms
	const slot = 100

	var era localstatequery.EraHistoryResult
	era.Begin.SlotNo = 0
	era.End.SlotNo = 0 // open-ended final era
	era.Params.EpochLength = 432000
	era.Params.SlotLength = slotLengthMs

	got, err := slotToPOSIXTime(
		slot,
		systemStartMs,
		[]localstatequery.EraHistoryResult{era},
	)
	if err != nil {
		t.Fatalf("slotToPOSIXTime() error = %v", err)
	}
	want := systemStartMs + int64(slot)*int64(slotLengthMs)
	if got != want {
		t.Fatalf("slotToPOSIXTime() = %d, want %d", got, want)
	}
}

// TestBuildTxInfoWithdrawalUsesStakingCredential verifies the withdrawals map
// key is a Plutus StakingCredential (StakingHash Credential =
// Constr 0 [Constr 0 [hash]]), not a bare Credential or address.
func TestBuildTxInfoWithdrawalUsesStakingCredential(t *testing.T) {
	stakeHash := bytes.Repeat([]byte{0xab}, 28)
	rewardAddr, err := lcommon.NewAddressFromParts(
		lcommon.AddressTypeNoneKey,
		lcommon.AddressNetworkMainnet,
		nil,
		stakeHash,
	)
	if err != nil {
		t.Fatalf("NewAddressFromParts() error = %v", err)
	}
	tx := &testTransaction{
		withdrawals: map[*lcommon.Address]*big.Int{
			&rewardAddr: big.NewInt(1_000_000),
		},
	}

	txInfo, err := buildTxInfoForVersion(
		tx,
		map[string]ledger.Utxo{},
		0,
		nil,
		0,
		plutusScriptV2,
	)
	if err != nil {
		t.Fatalf("buildTxInfoForVersion() error = %v", err)
	}
	txInfoData := txInfo.ToPlutusData()

	wdrl, ok := txInfoData.(*data.Constr).Fields[6].(*data.Map)
	if !ok {
		t.Fatalf(
			"withdrawals field is %T, want *data.Map",
			txInfoData.(*data.Constr).Fields[6],
		)
	}
	if len(wdrl.Pairs) != 1 {
		t.Fatalf("expected 1 withdrawal, got %d", len(wdrl.Pairs))
	}
	// StakingCredential = StakingHash Credential = Constr 0 [ Credential ]
	outer, ok := wdrl.Pairs[0][0].(*data.Constr)
	if !ok || outer.Tag != 0 || len(outer.Fields) != 1 {
		t.Fatalf("withdrawal key is not a StakingCredential constr: %#v", wdrl.Pairs[0][0])
	}
	// The inner field must itself be a Credential constr, not a raw bytestring.
	cred, ok := outer.Fields[0].(*data.Constr)
	if !ok {
		t.Fatalf(
			"withdrawal key wraps a %T, want a Credential constr (key encoded as bare credential/address)",
			outer.Fields[0],
		)
	}
	if cred.Tag != 0 || len(cred.Fields) != 1 { // PubKeyCredential = Constr 0 [hash]
		t.Fatalf("inner credential is not PubKeyCredential: %#v", cred)
	}
	hashBytes, ok := cred.Fields[0].(*data.ByteString)
	if !ok || !bytes.Equal(hashBytes.Inner, stakeHash) {
		t.Fatalf("credential hash mismatch: %#v", cred.Fields[0])
	}
}

// TestBuildTxInfoIncludesCertificates verifies certificates present in the
// transaction are encoded into the TxInfo dcert field instead of being dropped.
func TestBuildTxInfoIncludesCertificates(t *testing.T) {
	stakeHash := bytes.Repeat([]byte{0xcd}, 28)
	cert := &lcommon.StakeRegistrationCertificate{
		CertType: uint(lcommon.CertificateTypeStakeRegistration),
		StakeCredential: lcommon.Credential{
			CredType:   lcommon.CredentialTypeAddrKeyHash,
			Credential: lcommon.NewBlake2b224(stakeHash),
		},
	}
	tx := &testTransaction{
		certificates: []lcommon.Certificate{cert},
	}

	txInfo, err := buildTxInfoForVersion(
		tx,
		map[string]ledger.Utxo{},
		0,
		nil,
		0,
		plutusScriptV2,
	)
	if err != nil {
		t.Fatalf("buildTxInfoForVersion() error = %v", err)
	}
	txInfoData := txInfo.ToPlutusData()

	dcerts, ok := txInfoData.(*data.Constr).Fields[5].(*data.List)
	if !ok {
		t.Fatalf(
			"dcert field is %T, want *data.List",
			txInfoData.(*data.Constr).Fields[5],
		)
	}
	if len(dcerts.Items) != 1 {
		t.Fatalf("expected 1 dcert, got %d (certificates dropped)", len(dcerts.Items))
	}
	// DCertDelegRegKey = Constr 0 [StakingCredential]
	item, ok := dcerts.Items[0].(*data.Constr)
	if !ok || item.Tag != 0 {
		t.Fatalf("expected stake-registration DCert (Constr 0), got %#v", dcerts.Items[0])
	}
}

// TestBuildTxInfoRejectsUnsupportedCertificate verifies that a certificate type
// which cannot be represented in the legacy V1/V2 DCert format produces a clean
// error rather than a panic or a silently-dropped certificate.
func TestBuildTxInfoRejectsUnsupportedCertificate(t *testing.T) {
	tx := &testTransaction{
		certificates: []lcommon.Certificate{
			&lcommon.VoteDelegationCertificate{
				CertType: uint(lcommon.CertificateTypeVoteDelegation),
			},
		},
	}

	_, err := buildTxInfoForVersion(
		tx,
		map[string]ledger.Utxo{},
		0,
		nil,
		0,
		plutusScriptV2,
	)
	if err == nil {
		t.Fatal("expected error for unsupported certificate type")
	}
	if !strings.Contains(err.Error(), "certificate") {
		t.Fatalf("expected certificate error, got %q", err)
	}
}

func TestBuildScriptContextForVersionUsesLanguageSpecificTxInfo(
	t *testing.T,
) {
	tx := &testTransaction{}
	purpose := script.ScriptPurposeMinting{}
	redeemer := script.Redeemer{
		Tag:  lcommon.RedeemerTagMint,
		Data: data.NewConstr(0),
	}

	testCases := []struct {
		name              string
		version           plutusScriptVersion
		wantContextFields int
		wantTxInfoFields  int
	}{
		{
			name:              "v1",
			version:           plutusScriptV1,
			wantContextFields: 2,
			wantTxInfoFields:  10,
		},
		{
			name:              "v2",
			version:           plutusScriptV2,
			wantContextFields: 2,
			wantTxInfoFields:  12,
		},
		{
			name:              "v3",
			version:           plutusScriptV3,
			wantContextFields: 3,
			wantTxInfoFields:  16,
		},
		{
			name:              "v4",
			version:           plutusScriptV4,
			wantContextFields: 3,
			wantTxInfoFields:  16,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			contextData, err := buildScriptContextForVersion(
				tx,
				map[string]ledger.Utxo{},
				purpose,
				redeemer,
				0,
				nil,
				0,
				tc.version,
			)
			if err != nil {
				t.Fatalf("buildScriptContextForVersion() error = %v", err)
			}

			context, ok := contextData.(*data.Constr)
			if !ok {
				t.Fatalf("context is %T, want *data.Constr", contextData)
			}
			if len(context.Fields) != tc.wantContextFields {
				t.Fatalf(
					"context field count = %d, want %d",
					len(context.Fields),
					tc.wantContextFields,
				)
			}
			txInfo, ok := context.Fields[0].(*data.Constr)
			if !ok {
				t.Fatalf("txInfo is %T, want *data.Constr", context.Fields[0])
			}
			if len(txInfo.Fields) != tc.wantTxInfoFields {
				t.Fatalf(
					"txInfo field count = %d, want %d",
					len(txInfo.Fields),
					tc.wantTxInfoFields,
				)
			}
		})
	}
}

type testTransaction struct {
	lcommon.TransactionBodyBase
	inputs          []lcommon.TransactionInput
	referenceInputs []lcommon.TransactionInput
	withdrawals     map[*lcommon.Address]*big.Int
	certificates    []lcommon.Certificate
}

func (t testTransaction) Fee() *big.Int {
	return big.NewInt(0)
}

func (t testTransaction) Withdrawals() map[*lcommon.Address]*big.Int {
	return t.withdrawals
}

func (t testTransaction) Certificates() []lcommon.Certificate {
	return t.certificates
}

func (t testTransaction) Cbor() []byte {
	return nil
}

func (t testTransaction) Type() int {
	return 0
}

func (t testTransaction) Hash() lcommon.Blake2b256 {
	return lcommon.Blake2b256{}
}

func (t testTransaction) LeiosHash() lcommon.Blake2b256 {
	return lcommon.Blake2b256{}
}

func (t testTransaction) Metadata() lcommon.TransactionMetadatum {
	return nil
}

func (t testTransaction) AuxiliaryData() lcommon.AuxiliaryData {
	return nil
}

func (t testTransaction) IsValid() bool {
	return true
}

func (t testTransaction) Consumed() []lcommon.TransactionInput {
	return nil
}

func (t testTransaction) Produced() []lcommon.Utxo {
	return nil
}

func (t testTransaction) Witnesses() lcommon.TransactionWitnessSet {
	return nil
}

func (t testTransaction) Inputs() []lcommon.TransactionInput {
	return t.inputs
}

func (t testTransaction) ReferenceInputs() []lcommon.TransactionInput {
	return t.referenceInputs
}

func (t testTransaction) ProtocolParameterUpdates() (
	uint64,
	map[lcommon.Blake2b224]lcommon.ProtocolParameterUpdate,
) {
	return 0, nil
}

func testInput(suffix string, index int) lcommon.TransactionInput {
	return ledger.NewShelleyTransactionInput(strings.Repeat("00", 31)+suffix, index)
}
