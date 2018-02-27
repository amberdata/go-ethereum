package core

import (
	"strings"
	"testing"

	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
)

func TestSaveInternalTx(t *testing.T) {
	blockNumber, _ := big.NewInt(0).SetString("4370000", 10)
	timestamp, _ := big.NewInt(0).SetString("1508131331", 10)
	thash := common.HexToHash("0xba87b27b862ba1a638ae0b418bde7ad4bcc8ee86d69f9c0b2d1fd69524491f2e")
	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
	value, _ := big.NewInt(0).SetString("999999999999999999", 10)
	opcode := "CREATE"
	txType := 1
	saveInternalTxFromSingleTx(dbo, blockNumber, thash, []*vm.InternalTx{
		&vm.InternalTx{
			BlockNumberNumber: blockNumber.Uint64(),
			TimestampSec:      timestamp.Int64(),
			ThashString:       strings.ToLower(thash.Hex()),
			SrcString:         strings.ToLower(src.Hex()),
			DestString:        strings.ToLower(dest.Hex()),
			ValueString:       value.Text(10),
			Opcode:            opcode,
			TxType:            txType,
			Depth:             1,
			Nonce:             1,
			InputString:       "",
			CodeString:        "",
			InitialGas:        0,
			LeftOverGas:       0,
			RetString:         "",
			ErrString:         "",
		},
	})
	fmt.Println("please manually verify in database")
}

func TestSaveInternalTx_AllFieldsNotNull(t *testing.T) {
	blockNumber, _ := big.NewInt(0).SetString("4370000", 10)
	timestamp, _ := big.NewInt(0).SetString("1508131331", 10)
	thash := common.HexToHash("0xba87b27b862ba1a638ae0b418bde7ad4bcc8ee86d69f9c0b2d1fd69524491f2e")
	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
	value, _ := big.NewInt(0).SetString("999999999999999999", 10)
	opcode := "CREATE"
	txType := 1
	input := hexutil.MustDecode("0x01")
	code := hexutil.MustDecode("0x02")
	ret := hexutil.MustDecode("0x03")
	err := errors.New("err")
	saveInternalTxFromSingleTx(dbo, blockNumber, thash, []*vm.InternalTx{
		&vm.InternalTx{
			BlockNumberNumber: blockNumber.Uint64(),
			TimestampSec:      timestamp.Int64(),
			ThashString:       strings.ToLower(thash.Hex()),
			SrcString:         strings.ToLower(src.Hex()),
			DestString:        strings.ToLower(dest.Hex()),
			ValueString:       value.Text(10),
			Opcode:            opcode,
			TxType:            txType,
			Depth:             1,
			Nonce:             2,
			InputString:       hexutil.Encode(input),
			CodeString:        hexutil.Encode(code),
			InitialGas:        0,
			LeftOverGas:       0,
			RetString:         hexutil.Encode(ret),
			ErrString:         err.Error(),
		},
	})
	fmt.Println("please manually verify in database")
}
