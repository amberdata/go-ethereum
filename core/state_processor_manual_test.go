package core

import (
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
			BlockNumber: blockNumber,
			Timestamp:   timestamp,
			Thash:       thash,
			Src:         src,
			Dest:        dest,
			Value:       value,
			Opcode:      opcode,
			TxType:      txType,
			Depth:       1,
			Nonce:       1,
			Input:       nil,
			Code:        nil,
			InitialGas:  0,
			LeftOverGas: 0,
			Ret:         nil,
			Err:         nil,
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
			BlockNumber: blockNumber,
			Timestamp:   timestamp,
			Thash:       thash,
			Src:         src,
			Dest:        dest,
			Value:       value,
			Opcode:      opcode,
			TxType:      txType,
			Depth:       1,
			Nonce:       2,
			Input:       input,
			Code:        code,
			InitialGas:  0,
			LeftOverGas: 0,
			Ret:         ret,
			Err:         err,
		},
	})
	fmt.Println("please manually verify in database")
}
