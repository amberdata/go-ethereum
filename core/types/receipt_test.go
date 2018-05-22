// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestJsonMarshaling(t *testing.T) {
	blockNumber, _ := big.NewInt(0).SetString("4370000", 10)
	timestamp, _ := big.NewInt(0).SetString("1508131331", 10)
	thash := common.HexToHash("0xba87b27b862ba1a638ae0b418bde7ad4bcc8ee86d69f9c0b2d1fd69524491f2e")
	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
	contractCodeAddr := common.Address([common.AddressLength]byte{})
	value, _ := big.NewInt(0).SetString("18446744073709551616", 10)
	opcode := "CREATE"
	txType := 1

	internalTx := InternalTx{
		BlockNumberNumber:      blockNumber.Uint64(),
		TimestampSec:           timestamp.Int64(),
		ThashString:            strings.ToLower(thash.Hex()),
		SrcString:              strings.ToLower(src.Hex()),
		DestString:             strings.ToLower(dest.Hex()),
		ContractCodeAddrString: strings.ToLower(contractCodeAddr.Hex()),
		ValueBigInt:            value,
		Opcode:                 opcode,
		TxType:                 txType,
		Depth:                  1,
		Index:                  1,
		InputString:            "",
		CodeString:             "",
		InitialGas:             0,
		LeftOverGas:            0,
		RetString:              "",
		ErrString:              "",
	}
	internalTxMarshaled, _ := json.Marshal(internalTx)
	assert.Equal(t, `{"blockNumber":4370000,"timestamp":1508131331,"transactionHash":"0xba87b27b862ba1a638ae0b418bde7ad4bcc8ee86d69f9c0b2d1fd69524491f2e","from":"0xd3e32594cedbc102d739142aa70d21f4caea5618","to":"0x2213d4738bfec14a2f98df5e428f48ebbde33e12","contractCodeAddress":"0x0000000000000000000000000000000000000000","value":18446744073709551616,"opcode":"CREATE","transactionTypeId":1,"depth":1,"messageIndex":1,"input":"","code":"","initialGas":0,"leftOverGas":0,"returnValue":"","error":""}`, string(internalTxMarshaled))
}
