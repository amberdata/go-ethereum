package db

import (
	"fmt"
	"testing"

	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

func TestKafkaDatastoreSaveInternalTx_Single(t *testing.T) {
	blockNumber, _ := big.NewInt(0).SetString("4370000", 10)
	timestamp, _ := big.NewInt(0).SetString("1508131331", 10)
	thash := common.HexToHash("0xba87b27b862ba1a638ae0b418bde7ad4bcc8ee86d69f9c0b2d1fd69524491f2e")
	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
	contractCodeAddr := common.Address([common.AddressLength]byte{})
	// value, _ := big.NewInt(0).SetString("999999999999999999", 10)
	opcode := "CREATE"
	txType := 1
	newEVM := &vm.EVM{InternalTxStore: []*types.InternalTx{}}
	newEVM.SaveInternalTx(blockNumber, timestamp, thash, src, dest, contractCodeAddr, nil, opcode, txType, 1, 1, nil, nil, 0, 0, nil, nil)
	newKafkaDatastore := NewKafkaDatastore([]string{"localhost:9092"})
	newKafkaDatastore.SaveInternalTxFromSingleBlock(blockNumber, newEVM.InternalTxStore)
	fmt.Println("please manually verify in Kafka")
}

func TestKafkaDatastoreSaveInternalTx_Multiple(t *testing.T) {
	blockNumber, _ := big.NewInt(0).SetString("4370000", 10)
	timestamp, _ := big.NewInt(0).SetString("1508131331", 10)
	thash := common.HexToHash("0xba87b27b862ba1a638ae0b418bde7ad4bcc8ee86d69f9c0b2d1fd69524491f2e")
	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
	contractCodeAddr := common.HexToAddress("0x4b8d449d0ed83032fbe1d1bce6bb2b4ca4252f7f")
	value, _ := big.NewInt(0).SetString("999999999999999999", 10)
	opcode := "CREATE"
	txType := 1
	newEVM := &vm.EVM{InternalTxStore: []*types.InternalTx{}}
	newEVM.SaveInternalTx(blockNumber, timestamp, thash, src, dest, contractCodeAddr, value, opcode, txType, 1, 10, nil, nil, 0, 0, nil, nil)
	newEVM.SaveInternalTx(blockNumber, timestamp, thash, src, dest, contractCodeAddr, value, opcode, txType, 1, 11, nil, nil, 0, 0, nil, nil)
	newKafkaDatastore := NewKafkaDatastore([]string{"localhost:9092"})
	newKafkaDatastore.SaveInternalTxFromSingleBlock(blockNumber, newEVM.InternalTxStore)
	fmt.Println("please manually verify in Kafka")
}
