package vm

import (
	"testing"

	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestSaveInternalTx_Single(t *testing.T) {
	blockNumber, _ := big.NewInt(0).SetString("4370000", 10)
	timestamp, _ := big.NewInt(0).SetString("1508131331", 10)
	thash := common.HexToHash("0xba87b27b862ba1a638ae0b418bde7ad4bcc8ee86d69f9c0b2d1fd69524491f2e")
	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
	// value, _ := big.NewInt(0).SetString("999999999999999999", 10)
	opcode := "CREATE"
	txType := 1
	newEVM := &EVM{InternalTxStore: []*InternalTx{}}
	newEVM.SaveInternalTx(blockNumber, timestamp, thash, src, dest, nil, opcode, txType, 1, 1, nil, nil, 0, 0, nil, nil)
	assert.Equal(t, 1, len(newEVM.InternalTxStore))
	first := newEVM.InternalTxStore[0]
	assert.Equal(t, "0", first.ValueString)
	assert.Equal(t, "", first.InputString)
	assert.Equal(t, "", first.CodeString)
	assert.Equal(t, "", first.RetString)
	assert.Equal(t, "", first.ErrString)
}

func TestSaveInternalTx_Multiple(t *testing.T) {
	blockNumber, _ := big.NewInt(0).SetString("4370000", 10)
	timestamp, _ := big.NewInt(0).SetString("1508131331", 10)
	thash := common.HexToHash("0xba87b27b862ba1a638ae0b418bde7ad4bcc8ee86d69f9c0b2d1fd69524491f2e")
	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
	value, _ := big.NewInt(0).SetString("999999999999999999", 10)
	opcode := "CREATE"
	txType := 1
	newEVM := &EVM{InternalTxStore: []*InternalTx{}}
	newEVM.SaveInternalTx(blockNumber, timestamp, thash, src, dest, value, opcode, txType, 1, 1, nil, nil, 0, 0, nil, nil)
	newEVM.SaveInternalTx(blockNumber, timestamp, thash, src, dest, value, opcode, txType, 1, 1, nil, nil, 0, 0, nil, nil)
	assert.Equal(t, 2, len(newEVM.InternalTxStore))
}

func TestSaveInternalTx_NonceShouldIncrease(t *testing.T) {
	blockNumber, _ := big.NewInt(0).SetString("4370000", 10)
	timestamp, _ := big.NewInt(0).SetString("1508131331", 10)
	thash := common.HexToHash("0xba87b27b862ba1a638ae0b418bde7ad4bcc8ee86d69f9c0b2d1fd69524491f2e")
	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
	value, _ := big.NewInt(0).SetString("999999999999999999", 10)
	opcode := "CREATE"
	txType := 1
	newEVM := &EVM{InternalTxStore: []*InternalTx{}}
	initialNonce := uint64(10)
	newEVM.InternalTxNonce = initialNonce
	newEVM.SaveInternalTx(blockNumber, timestamp, thash, src, dest, value, opcode, txType, 1, newEVM.InternalTxNonce, nil, nil, 0, 0, nil, nil)
	assert.Equal(t, initialNonce+1, newEVM.InternalTxNonce)
}

// func TestSaveInternalTx2(t *testing.T) {
// 	blockNumber, _ := big.NewInt(0).SetString("4370000", 10)
// 	timestamp, _ := big.NewInt(0).SetString("1508131331", 10)
// 	thash := common.HexToHash("0xba87b27b862ba1a638ae0b418bde7ad4bcc8ee86d69f9c0b2d1fd69524491f2d")
// 	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
// 	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
// 	value, _ := big.NewInt(0).SetString("999999999999999999", 10)
// 	opcode := "CREATE"
// 	txType := 1
// 	newEVM := &EVM{}
// 	input := hexutil.MustDecode("0x01")
// 	code := hexutil.MustDecode("0x02")
// 	ret := hexutil.MustDecode("0x03")
// 	err := errors.New("err")
// 	newEVM.SaveInternalTx(blockNumber, timestamp, thash, src, dest, value, opcode, txType, 1, 2, input, code, 0, 0, ret, err)
// 	fmt.Println("please manually verify in database")
// }
