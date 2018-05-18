package core

import (
	"strings"
	"testing"

	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/db"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestSaveInternalTxFromSingleBlock(t *testing.T) {
	blockNumber, _ := big.NewInt(0).SetString("4370000", 10)
	timestamp, _ := big.NewInt(0).SetString("1508131331", 10)
	thash := common.HexToHash("0x56a975e3fed5a1b8a246a94987ba61be5a1c96086076d90f1b5f2c5fb9af1bc7")
	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
	contractCodeAddr := common.Address([common.AddressLength]byte{})
	value, _ := big.NewInt(0).SetString("999999999999999999999999999999999999", 10)
	opcode := "CREATE"
	txType := 1
	saveInternalTxFromSingleBlock(db.DBO, blockNumber, []*types.InternalTx{
		&types.InternalTx{
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
		},
	})
	fmt.Println("please manually verify in database")
}

func TestSaveInternalTxFromSingleBlock_AllFieldsNotNull(t *testing.T) {
	blockNumber, _ := big.NewInt(0).SetString("4370000", 10)
	timestamp, _ := big.NewInt(0).SetString("1508131331", 10)
	thash := common.HexToHash("0x56a975e3fed5a1b8a246a94987ba61be5a1c96086076d90f1b5f2c5fb9af1bc7")
	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
	contractCodeAddr := common.HexToAddress("0x4b8d449d0ed83032fbe1d1bce6bb2b4ca4252f7f")

	value, _ := big.NewInt(0).SetString("999999999999999999", 10)
	opcode := "CREATE"
	txType := 1
	input := hexutil.MustDecode("0x01")
	code := hexutil.MustDecode("0x02")
	ret := hexutil.MustDecode("0x03")
	err := errors.New("err")
	saveInternalTxFromSingleBlock(db.DBO, blockNumber, []*types.InternalTx{
		&types.InternalTx{
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
			Index:                  2,
			InputString:            hexutil.Encode(input),
			CodeString:             hexutil.Encode(code),
			InitialGas:             0,
			LeftOverGas:            0,
			RetString:              hexutil.Encode(ret),
			ErrString:              err.Error(),
		},
	})
	fmt.Println("please manually verify in database")
}

//
// func TestSaveInternalTxFromSingleTx(t *testing.T) {
// 	blockNumber, _ := big.NewInt(0).SetString("4370000", 10)
// 	timestamp, _ := big.NewInt(0).SetString("1508131331", 10)
// 	thash := common.HexToHash("0xba87b27b862ba1a638ae0b418bde7ad4bcc8ee86d69f9c0b2d1fd69524491f2e")
// 	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
// 	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
// 	contractCodeAddr := common.Address([common.AddressLength]byte{})
// 	value, _ := big.NewInt(0).SetString("999999999999999999", 10)
// 	opcode := "CREATE"
// 	txType := 1
// 	saveInternalTxFromSingleTx(dbo, blockNumber, thash, []*types.InternalTx{
// 		&types.InternalTx{
// 			BlockNumberNumber:      blockNumber.Uint64(),
// 			TimestampSec:           timestamp.Int64(),
// 			ThashString:            strings.ToLower(thash.Hex()),
// 			SrcString:              strings.ToLower(src.Hex()),
// 			DestString:             strings.ToLower(dest.Hex()),
// 			ContractCodeAddrString: strings.ToLower(contractCodeAddr.Hex()),
// 			ValueString:            value.Text(10),
// 			Opcode:                 opcode,
// 			TxType:                 txType,
// 			Depth:                  1,
// 			Index:                  1,
// 			InputString:            "",
// 			CodeString:             "",
// 			InitialGas:             0,
// 			LeftOverGas:            0,
// 			RetString:              "",
// 			ErrString:              "",
// 		},
// 	})
// 	fmt.Println("please manually verify in database")
// }
//
// func TestSaveInternalTxFromSingleTx_AllFieldsNotNull(t *testing.T) {
// 	blockNumber, _ := big.NewInt(0).SetString("4370000", 10)
// 	timestamp, _ := big.NewInt(0).SetString("1508131331", 10)
// 	thash := common.HexToHash("0xba87b27b862ba1a638ae0b418bde7ad4bcc8ee86d69f9c0b2d1fd69524491f2e")
// 	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
// 	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
// 	contractCodeAddr := common.HexToAddress("0x4b8d449d0ed83032fbe1d1bce6bb2b4ca4252f7f")
//
// 	value, _ := big.NewInt(0).SetString("999999999999999999", 10)
// 	opcode := "CREATE"
// 	txType := 1
// 	input := hexutil.MustDecode("0x01")
// 	code := hexutil.MustDecode("0x02")
// 	ret := hexutil.MustDecode("0x03")
// 	err := errors.New("err")
// 	saveInternalTxFromSingleTx(dbo, blockNumber, thash, []*types.InternalTx{
// 		&types.InternalTx{
// 			BlockNumberNumber:      blockNumber.Uint64(),
// 			TimestampSec:           timestamp.Int64(),
// 			ThashString:            strings.ToLower(thash.Hex()),
// 			SrcString:              strings.ToLower(src.Hex()),
// 			DestString:             strings.ToLower(dest.Hex()),
// 			ContractCodeAddrString: strings.ToLower(contractCodeAddr.Hex()),
// 			ValueString:            value.Text(10),
// 			Opcode:                 opcode,
// 			TxType:                 txType,
// 			Depth:                  1,
// 			Index:                  2,
// 			InputString:            hexutil.Encode(input),
// 			CodeString:             hexutil.Encode(code),
// 			InitialGas:             0,
// 			LeftOverGas:            0,
// 			RetString:              hexutil.Encode(ret),
// 			ErrString:              err.Error(),
// 		},
// 	})
// 	fmt.Println("please manually verify in database")
// }

// func TestSaveInternalTx_LargeAmountData(t *testing.T) {
// 	blockNumber, _ := big.NewInt(0).SetString("2409161", 10)
// 	timestamp, _ := big.NewInt(0).SetString("1508131331", 10)
// 	thash := common.HexToHash("0x5713c8444f1ff69b13ee49dda033a777591046314ea5e76bd278fb8af7e3c96b")
// 	src := common.HexToAddress("0xd3e32594cedbc102d739142aa70d21f4caea5618")
// 	dest := common.HexToAddress("0x2213d4738bfec14a2f98df5e428f48ebbde33e12")
// 	value, _ := big.NewInt(0).SetString("999999999999999999", 10)
// 	opcode := "CREATE"
// 	txType := 1
// 	input := hexutil.MustDecode("0x01")
// 	code := hexutil.MustDecode("0x02")
// 	ret := hexutil.MustDecode("0x03")
// 	err := errors.New("err")
// 	internalTx := []*types.InternalTx{}
// 	for i := 0; i < 17779; i++ {
// 		internalTx = append(internalTx, &types.InternalTx{
// 			BlockNumberNumber: blockNumber.Uint64(),
// 			TimestampSec:      timestamp.Int64(),
// 			ThashString:       strings.ToLower(thash.Hex()),
// 			SrcString:         strings.ToLower(src.Hex()),
// 			DestString:        strings.ToLower(dest.Hex()),
// 			ValueString:       value.Text(10),
// 			Opcode:            opcode,
// 			TxType:            txType,
// 			Depth:             1,
// 			Index:             uint64(i),
// 			InputString:       hexutil.Encode(input),
// 			CodeString:        hexutil.Encode(code),
// 			InitialGas:        0,
// 			LeftOverGas:       0,
// 			RetString:         hexutil.Encode(ret),
// 			ErrString:         err.Error(),
// 		})
// 	}
// 	saveInternalTxFromSingleBlock(dbo, blockNumber, internalTx)
// 	fmt.Println("please manually verify in database")
// }
