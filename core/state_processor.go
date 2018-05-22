// Copyright 2015 The go-ethereum Authors
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

package core

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/db"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var enableSaveInternalTxSql = getEnableSaveInternalTxSql()

func getEnableSaveInternalTxSql() bool {
	fmt.Printf("flag.Lookup(\"test.v\") = %s\n", flag.Lookup("test.v")) // too strange: if this line is removed, one test will fail!
	if flag.Lookup("test.v") != nil {
		return true
	}
	enableSaveInternalTxSqlString := os.Getenv("GETH_ENABLE_SAVE_INTERNAL_MESSAGE")
	if len(enableSaveInternalTxSqlString) > 0 {
		enableSaveInternalTxSql, err := strconv.ParseBool(enableSaveInternalTxSqlString)
		if err != nil {
			panic(fmt.Sprintf("Cannot parse enableSaveInternalTxSqlString: %s", enableSaveInternalTxSqlString))
		}
		fmt.Printf("enableSaveInternalTxSql = %t\n", enableSaveInternalTxSql)
		return enableSaveInternalTxSql
	} else {
		return false
	}
}

var fullSyncEndBlock = getFullSyncEndBlock()

func getFullSyncEndBlock() uint64 {
	fmt.Printf("flag.Lookup(\"test.v\") = %s\n", flag.Lookup("test.v")) // too strange: if this line is removed, one test will fail!
	if flag.Lookup("test.v") != nil {
		return 0
	}
	fullSyncEndBlockString := os.Getenv("GETH_FULL_SYNC_END_BLOCK")
	if len(fullSyncEndBlockString) > 0 {
		fullSyncEndBlock, err := strconv.ParseUint(fullSyncEndBlockString, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("Cannot parse fullSyncEndBlockString: %s", fullSyncEndBlockString))
		}
		fmt.Printf("fullSyncEndBlock = %d\n", fullSyncEndBlock)
		return fullSyncEndBlock
	} else {
		return 0
	}
}

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for block rewards
	ds     db.Datastore
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		engine: engine,
		ds:     db.NewKafkaDatastore([]string{os.Getenv("KAFKA_HOSTNAME") + ":" + os.Getenv("KAFKA_PORT")}),
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, *big.Int, error) {
	if fullSyncEndBlock > 0 && block.NumberU64() > fullSyncEndBlock {
		panic(fmt.Sprintf("going beyond fullSyncEndBlock: block.NumberU64() = %d, fullSyncEndBlock = %d", block.NumberU64(), fullSyncEndBlock))
	}
	if flag.Lookup("test.v") == nil {
		shouldWait := true
		for shouldWait {
			var maxBlockNumber uint64
			// err := db.DBO.QueryRow(`SELECT MAX(number) FROM block`).Scan(&maxBlockNumber)
			// common.CheckErr(err, nil)
			var dummyBlockNumber uint64
			var epoch1, epoch2 int64
			func() {
				blockRows, err1 := db.DBO.Query(`SELECT number, EXTRACT(EPOCH FROM timestamp)::BIGINT FROM block ORDER BY number DESC LIMIT 2`)
				defer blockRows.Close()
				common.CheckErr(err1, nil)
				if blockRows.Next() {
					err2 := blockRows.Scan(&maxBlockNumber, &epoch2)
					common.CheckErr(err2, nil)
				}
				if blockRows.Next() {
					err3 := blockRows.Scan(&dummyBlockNumber, &epoch1)
					common.CheckErr(err3, nil)
				}
			}()

			blockTime := uint64(epoch2 - epoch1)
			// fmt.Printf("blockTime = %d seconds\n", blockTime)
			// fmt.Printf("maxBlockNumber = %d\n", maxBlockNumber)
			// fmt.Printf("block.NumberU64() = %d\n", block.NumberU64())
			// fmt.Printf("block.NumberU64()+6 = %d\n", block.NumberU64()+6)
			if maxBlockNumber <= block.NumberU64()+6 {
				fmt.Printf("blockTime = %d seconds, go to sleep\n", blockTime)
				time.Sleep(time.Duration(blockTime) * time.Second)
			} else {
				shouldWait = false
			}
		}
	}

	var (
		receipts     types.Receipts
		totalUsedGas = big.NewInt(0)
		header       = block.Header()
		allLogs      []*types.Log
		gp           = new(GasPool).AddGas(block.GasLimit())
	)
	// Mutate the the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	allInternalTxs := []*types.InternalTx{}
	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		statedb.Prepare(tx.Hash(), block.Hash(), i)
		receipt, _, err := ApplyTransaction(p.config, p.bc, nil, gp, statedb, header, tx, totalUsedGas, cfg)
		if err != nil {
			return nil, nil, nil, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
		allInternalTxs = append(allInternalTxs, receipt.InternalTxStore...)
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles(), receipts)
	if flag.Lookup("test.v") == nil {
		if shouldSaveInternalTxFromSingleBlock(db.DBO, block) {
			saveInternalTxFromSingleBlock(db.DBO, block.Number(), allInternalTxs)
			p.ds.SaveInternalTxFromSingleBlock(block.Number(), allInternalTxs)
		}
	}
	log.Info(fmt.Sprintf("Processed block %d, timestamp %s, hash %s, td %s", block.NumberU64(), time.Unix(block.Time().Int64(), 0).UTC().String(), block.Hash().Hex(), block.DeprecatedTd().String()))
	return receipts, allLogs, totalUsedGas, nil
}

func shouldSaveInternalTxFromSingleBlock(dbo *sql.DB, block *types.Block) bool {
	var canonicalHash string
	err := dbo.QueryRow(`SELECT hash FROM block WHERE number = $1`, block.NumberU64()).Scan(&canonicalHash)
	common.CheckErr(err, nil)
	return canonicalHash == strings.ToLower(block.Hash().Hex())
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *params.ChainConfig, bc *BlockChain, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *big.Int, cfg vm.Config) (*types.Receipt, *big.Int, error) {
	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number))
	if err != nil {
		return nil, nil, err
	}
	// Create a new context to be used in the EVM environment
	context := NewEVMContext(msg, header, bc, author)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmenv := vm.NewEVM(context, statedb, config, cfg)
	// Apply the transaction to the current state (included in the env)
	_, gas, failed, err := ApplyMessage(vmenv, msg, gp)
	if err != nil {
		return nil, nil, err
	}

	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	usedGas.Add(usedGas, gas)

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing wether the root touch-delete accounts.
	receipt := types.NewReceipt(root, failed, usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = new(big.Int).Set(gas)
	// if the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, tx.Nonce())
	}

	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	// saveInternalTxFromSingleTx(dbo, vmenv.BlockNumber, statedb.GetThash(), vmenv.InternalTxStore)
	receipt.InternalTxStore = vmenv.InternalTxStore
	return receipt, gas, err
}

func saveInternalTxFromSingleBlock(dbo *sql.DB, blockNumber *big.Int, internalTxStore []*types.InternalTx) uint64 {
	if !enableSaveInternalTxSql {
		return 0
	}
	if len(internalTxStore) == 0 {
		return 0
	}
	// fmt.Printf("len(internalTxStore) = %d\n", len(internalTxStore))
	startTimestamp := time.Now().UTC()
	var buffer bytes.Buffer
	for _, internalTx := range internalTxStore {
		buffer.WriteString("('")
		buffer.WriteString(strconv.FormatUint(internalTx.BlockNumberNumber, 10))
		buffer.WriteString("',to_timestamp(")
		buffer.WriteString(strconv.FormatInt(int64(time.Unix(internalTx.TimestampSec, 0).UTC().Unix()), 10))
		buffer.WriteString("),'")
		buffer.WriteString(internalTx.ThashString)
		buffer.WriteString("','")
		buffer.WriteString(internalTx.SrcString)
		buffer.WriteString("','")
		buffer.WriteString(internalTx.DestString)
		buffer.WriteString("','")
		buffer.WriteString(internalTx.ContractCodeAddrString)
		buffer.WriteString("','")
		buffer.WriteString(internalTx.ValueBigInt.Text(10))
		buffer.WriteString("','")
		buffer.WriteString(internalTx.Opcode)
		buffer.WriteString("','")
		buffer.WriteString(strconv.FormatInt(int64(internalTx.TxType), 10))
		buffer.WriteString("','")
		buffer.WriteString(strconv.FormatInt(int64(internalTx.Depth), 10))
		buffer.WriteString("','")
		buffer.WriteString(strconv.FormatUint(internalTx.Index, 10))
		buffer.WriteString("','")
		buffer.WriteString(internalTx.InputString)
		buffer.WriteString("','")
		buffer.WriteString(internalTx.CodeString)
		buffer.WriteString("','")
		buffer.WriteString(strconv.FormatUint(internalTx.InitialGas, 10))
		buffer.WriteString("','")
		buffer.WriteString(strconv.FormatUint(internalTx.LeftOverGas, 10))
		buffer.WriteString("','")
		buffer.WriteString(internalTx.RetString)
		buffer.WriteString("','")
		buffer.WriteString(internalTx.ErrString)
		buffer.WriteString("'),")
	}
	sqlStr := fmt.Sprintf(`
		INSERT INTO internal_message (
		"blockNumber",
		"timestamp",
		"transactionHash",
		"from",
		"to",
		"contractCodeAddress",
		"value",
		"opcode",
		"transactionTypeId",
		"depth",
		"messageIndex",
		"input",
		"code",
		"initialGas",
		"leftOverGas",
		"returnValue",
		"error"
		)
		VALUES %s
		ON CONFLICT ("transactionHash", "messageIndex")
		DO UPDATE SET
		"blockNumber" = EXCLUDED."blockNumber", "timestamp" = EXCLUDED."timestamp"
		`,
		strings.TrimSuffix(buffer.String(), ","))
	// fmt.Printf("sqlStr = %s\n", sqlStr)
	result, err1 := dbo.Exec(sqlStr)
	common.CheckErr(err1, nil)
	totalRowsAffected, err2 := result.RowsAffected()
	common.CheckErr(err2, nil)
	endTimestamp := time.Now().UTC()
	elapsed := endTimestamp.Sub(startTimestamp)
	fmt.Printf("%s: execution took %s, saved internal tx: blockNumber = %d\n", endTimestamp.Format("2006-01-02 15:04:05"), elapsed.Round(time.Millisecond).String(), blockNumber.Uint64())
	fmt.Printf("len(internalTxStore) = %d, totalRowsAffected = %d\n", len(internalTxStore), totalRowsAffected)
	return uint64(totalRowsAffected)
}

// func saveInternalTxFromSingleBlockRequired(dbo *sql.DB, blockNumber *big.Int, internalTxStore []*types.InternalTx, deleteFirst bool) uint64 {
// 	if len(internalTxStore) == 0 {
// 		return 0
// 	}
// 	fmt.Printf("len(internalTxStore) = %d\n", len(internalTxStore))
// 	startTimestamp := time.Now().UTC()
// 	txn, err1 := dbo.Begin()
// 	common.CheckErr(err1, txn)
// 	fmt.Println("no err1")
// 	if deleteFirst {
// 		_, err2 := txn.Exec(`DELETE FROM internal_message WHERE "blockNumber" = $1`, blockNumber.Uint64())
// 		common.CheckErr(err2, txn)
// 		fmt.Println("deleteFirst is true and no err2")
// 	}
// 	stmt, err2 := txn.Prepare(pq.CopyIn("internal_message", "blockNumber", "timestamp", "transactionHash", "from", "to", "contractCodeAddress", "value", "opcode", "transactionTypeId", "depth", "messageIndex", "input", "code", "initialGas", "leftOverGas", "returnValue", "error"))
// 	common.CheckErr(err2, txn)
// 	fmt.Println("no err2")
// 	for _, internalTx := range internalTxStore {
// 		_, err := stmt.Exec(internalTx.BlockNumberNumber, time.Unix(internalTx.TimestampSec, 0).UTC(), internalTx.ThashString, internalTx.SrcString, internalTx.DestString, internalTx.ContractCodeAddrString, internalTx.ValueString, internalTx.Opcode, internalTx.TxType, internalTx.Depth, internalTx.Index, internalTx.InputString, internalTx.CodeString, internalTx.InitialGas, internalTx.LeftOverGas, internalTx.RetString, internalTx.ErrString)
// 		common.CheckErr(err, txn)
// 	}
// 	fmt.Println("no error in the loop of stmt.Exec(internalTx.BlockNumberNumber...")
// 	_, err3 := stmt.Exec()
// 	common.CheckErr(err3, txn)
// 	fmt.Println("no err3")
// 	// err4 := stmt.Close()
// 	// common.CheckErr(err4, txn)
// 	err5 := txn.Commit()
// 	common.CheckErr(err5, txn)
// 	fmt.Println("no err5")
// 	endTimestamp := time.Now().UTC()
// 	elapsed := endTimestamp.Sub(startTimestamp)
// 	fmt.Printf("%s: execution took %s, saved internal tx: blockNumber = %d\n", endTimestamp.Format("2006-01-02 15:04:05"), elapsed.Round(time.Millisecond).String(), blockNumber.Uint64())
// 	return uint64(len(internalTxStore))
// }

// func saveInternalTxFromSingleTx(dbo *sql.DB, blockNumber *big.Int, tHash common.Hash, internalTxStore []*types.InternalTx) uint64 {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			fmt.Println("Recovered in saveInternalTxFromSingleTx: ", r)
// 			saveInternalTx(dbo, internalTxStore)
// 		}
// 	}()
// 	if len(internalTxStore) == 0 {
// 		return 0
// 	}
// 	fmt.Printf("len(internalTxStore) = %d\n", len(internalTxStore))
// 	startTimestamp := time.Now().UTC()
// 	txn, err1 := dbo.Begin()
// 	common.CheckErr(err1, txn)
// 	stmt, err2 := txn.Prepare(pq.CopyIn("internal_message", "blockNumber", "timestamp", "transactionHash", "from", "to", "contractCodeAddress", "value", "opcode", "transactionTypeId", "depth", "messageIndex", "input", "code", "initialGas", "leftOverGas", "returnValue", "error"))
// 	common.CheckErr(err2, txn)
// 	for _, internalTx := range internalTxStore {
// 		_, err := stmt.Exec(internalTx.BlockNumberNumber, time.Unix(internalTx.TimestampSec, 0).UTC(), internalTx.ThashString, internalTx.SrcString, internalTx.DestString, internalTx.ContractCodeAddrString, internalTx.ValueString, internalTx.Opcode, internalTx.TxType, internalTx.Depth, internalTx.Index, internalTx.InputString, internalTx.CodeString, internalTx.InitialGas, internalTx.LeftOverGas, internalTx.RetString, internalTx.ErrString)
// 		common.CheckErr(err, txn)
// 	}
// 	_, err3 := stmt.Exec()
// 	common.CheckErr(err3, txn)
// 	// err4 := stmt.Close()
// 	// common.CheckErr(err4, txn)
// 	err5 := txn.Commit()
// 	common.CheckErr(err5, txn)
// 	endTimestamp := time.Now().UTC()
// 	elapsed := endTimestamp.Sub(startTimestamp)
// 	fmt.Printf("%s: execution took %s, saved internal tx: blockNumber = %d, transactionHash = %s\n", endTimestamp.Format("2006-01-02 15:04:05"), elapsed.Round(time.Millisecond).String(), blockNumber.Uint64(), strings.ToLower(tHash.Hex()))
// 	return uint64(len(internalTxStore))
// }

// func saveInternalTx(dbo *sql.DB, internalTxStore []*types.InternalTx) uint64 {
// 	totalRowsAffected := uint64(0)
// 	for _, internalTx := range internalTxStore {
// 		result, err2 := dbo.Exec("INSERT INTO internal_message (\"blockNumber\", \"timestamp\", \"transactionHash\", \"from\", \"to\", \"contractCodeAddress\",\"value\", \"opcode\", \"transactionTypeId\", \"depth\", \"messageIndex\", \"input\", \"code\", \"initialGas\", \"leftOverGas\", \"returnValue\", \"error\") VALUES ($1, $2, $3, $4, $5, $6, $7::NUMERIC, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17) ON CONFLICT (\"transactionHash\", \"messageIndex\") DO UPDATE SET \"blockNumber\" = EXCLUDED.\"blockNumber\", \"timestamp\" = EXCLUDED.\"timestamp\"",
// 			internalTx.BlockNumberNumber, time.Unix(internalTx.TimestampSec, 0).UTC(), internalTx.ThashString, internalTx.SrcString, internalTx.DestString, internalTx.ContractCodeAddrString, internalTx.ValueString, internalTx.Opcode, internalTx.TxType, internalTx.Depth, internalTx.Index, internalTx.InputString, internalTx.CodeString, internalTx.InitialGas, internalTx.LeftOverGas, internalTx.RetString, internalTx.ErrString)
// 		common.CheckErr(err2, nil)
// 		rowAffected, err3 := result.RowsAffected()
// 		common.CheckErr(err3, nil)
// 		// fmt.Printf("rowsAffected = %d\n", rowsAffected)
// 		timeNowString := time.Now().UTC().Format("2006-01-02 15:04:05")
// 		// if rowsAffected == 0 {
// 		severity := ""
// 		if rowAffected == 0 {
// 			severity = "warning"
// 		}
// 		fmt.Printf("%s %s: rowAffected == %d, blockNumber = %d, transactionHash = %s, index = %d\n", timeNowString, severity, rowAffected, internalTx.BlockNumberNumber, internalTx.ThashString, internalTx.Index)
// 		totalRowsAffected += uint64(rowAffected)
// 	}
// 	fmt.Printf("len(internalTxStore) = %d, totalRowsAffected = %d\n", len(internalTxStore), totalRowsAffected)
// 	return totalRowsAffected
// }
