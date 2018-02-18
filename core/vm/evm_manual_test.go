package vm

import (
	"testing"

	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestSaveInternalTx(t *testing.T) {
	thash := common.HexToHash("0xba87b27b862ba1a638ae0b418bde7ad4bcc8ee86d69f9c0b2d1fd69524491f2d")
	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
	value, _ := big.NewInt(0).SetString("999999999999999999", 10)
	opcode := "CREATE"
	txType := 1
	newEVM := &EVM{}
	newEVM.SaveInternalTx(thash, src, dest, value, opcode, txType, 1, 1, nil, nil, 0, 0, nil, nil)
	fmt.Println("please manually verify in database")
}

func TestSaveInternalTx2(t *testing.T) {
	thash := common.HexToHash("0xba87b27b862ba1a638ae0b418bde7ad4bcc8ee86d69f9c0b2d1fd69524491f2d")
	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
	value, _ := big.NewInt(0).SetString("999999999999999999", 10)
	opcode := "CREATE"
	txType := 1
	newEVM := &EVM{}
	input := hexutil.MustDecode("0x01")
	code := hexutil.MustDecode("0x02")
	ret := hexutil.MustDecode("0x03")
	err := errors.New("err")
	newEVM.SaveInternalTx(thash, src, dest, value, opcode, txType, 1, 2, input, code, 0, 0, ret, err)
	fmt.Println("please manually verify in database")
}
