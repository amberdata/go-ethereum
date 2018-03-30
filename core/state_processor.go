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
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/lib/pq"
)

var connStr = "host=" + os.Getenv("DATABASE_HOSTNAME") + " port=" + os.Getenv("DATABASE_PORT") + " dbname=" + os.Getenv("DATABASE_NAME") + " user=" + os.Getenv("DATABASE_USERNAME") + " password=" + os.Getenv("DATABASE_PASSWORD") + " sslmode=disable"
var dbo = connectDB(connStr)

func connectDB(connStr string) *sql.DB {
	dbo, err := sql.Open("postgres", connStr)
	common.CheckErr(err, nil)
	return dbo
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
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		engine: engine,
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
	saveInternalTxFromSingleBlock(dbo, block.Number(), allInternalTxs)
	log.Info(fmt.Sprintf("Processed block %d", block.NumberU64()))
	return receipts, allLogs, totalUsedGas, nil
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
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in saveInternalTxFromSingleBlock: ", r)
			saveInternalTx(dbo, internalTxStore)
		}
	}()
	if len(internalTxStore) == 0 {
		return 0
	}
	fmt.Printf("len(internalTxStore) = %d\n", len(internalTxStore))
	startTimestamp := time.Now().UTC()
	txn, err1 := dbo.Begin()
	common.CheckErr(err1, txn)
	stmt, err2 := txn.Prepare(pq.CopyIn("internal_message", "blockNumber", "timestamp", "transactionHash", "from", "to", "contractCodeAddress", "value", "opcode", "transactionTypeId", "depth", "messageIndex", "input", "code", "initialGas", "leftOverGas", "returnValue", "error"))
	common.CheckErr(err2, txn)
	for _, internalTx := range internalTxStore {
		_, err := stmt.Exec(internalTx.BlockNumberNumber, time.Unix(internalTx.TimestampSec, 0).UTC(), internalTx.ThashString, internalTx.SrcString, internalTx.DestString, internalTx.ContractCodeAddrString, internalTx.ValueString, internalTx.Opcode, internalTx.TxType, internalTx.Depth, internalTx.Index, internalTx.InputString, internalTx.CodeString, internalTx.InitialGas, internalTx.LeftOverGas, internalTx.RetString, internalTx.ErrString)
		common.CheckErr(err, txn)
	}
	_, err3 := stmt.Exec()
	common.CheckErr(err3, txn)
	// err4 := stmt.Close()
	// common.CheckErr(err4, txn)
	err5 := txn.Commit()
	common.CheckErr(err5, txn)
	endTimestamp := time.Now().UTC()
	elapsed := endTimestamp.Sub(startTimestamp)
	fmt.Printf("%s: execution took %s, saved internal tx: blockNumber = %d\n", endTimestamp.Format("2006-01-02 15:04:05"), elapsed.Round(time.Millisecond).String(), blockNumber.Uint64())
	return uint64(len(internalTxStore))
}

func saveInternalTxFromSingleTx(dbo *sql.DB, blockNumber *big.Int, tHash common.Hash, internalTxStore []*types.InternalTx) uint64 {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in saveInternalTxFromSingleTx: ", r)
			saveInternalTx(dbo, internalTxStore)
		}
	}()
	if len(internalTxStore) == 0 {
		return 0
	}
	fmt.Printf("len(internalTxStore) = %d\n", len(internalTxStore))
	startTimestamp := time.Now().UTC()
	txn, err1 := dbo.Begin()
	common.CheckErr(err1, txn)
	stmt, err2 := txn.Prepare(pq.CopyIn("internal_message", "blockNumber", "timestamp", "transactionHash", "from", "to", "contractCodeAddress", "value", "opcode", "transactionTypeId", "depth", "messageIndex", "input", "code", "initialGas", "leftOverGas", "returnValue", "error"))
	common.CheckErr(err2, txn)
	for _, internalTx := range internalTxStore {
		_, err := stmt.Exec(internalTx.BlockNumberNumber, time.Unix(internalTx.TimestampSec, 0).UTC(), internalTx.ThashString, internalTx.SrcString, internalTx.DestString, internalTx.ContractCodeAddrString, internalTx.ValueString, internalTx.Opcode, internalTx.TxType, internalTx.Depth, internalTx.Index, internalTx.InputString, internalTx.CodeString, internalTx.InitialGas, internalTx.LeftOverGas, internalTx.RetString, internalTx.ErrString)
		common.CheckErr(err, txn)
	}
	_, err3 := stmt.Exec()
	common.CheckErr(err3, txn)
	// err4 := stmt.Close()
	// common.CheckErr(err4, txn)
	err5 := txn.Commit()
	common.CheckErr(err5, txn)
	endTimestamp := time.Now().UTC()
	elapsed := endTimestamp.Sub(startTimestamp)
	fmt.Printf("%s: execution took %s, saved internal tx: blockNumber = %d, transactionHash = %s\n", endTimestamp.Format("2006-01-02 15:04:05"), elapsed.Round(time.Millisecond).String(), blockNumber.Uint64(), strings.ToLower(tHash.Hex()))
	return uint64(len(internalTxStore))
}

func saveInternalTx(dbo *sql.DB, internalTxStore []*types.InternalTx) uint64 {
	totalRowsAffected := uint64(0)
	for _, internalTx := range internalTxStore {
		result, err2 := dbo.Exec("INSERT INTO internal_message (\"blockNumber\", \"timestamp\", \"transactionHash\", \"from\", \"to\", \"contractCodeAddress\",\"value\", \"opcode\", \"transactionTypeId\", \"depth\", \"messageIndex\", \"input\", \"code\", \"initialGas\", \"leftOverGas\", \"returnValue\", \"error\") VALUES ($1, $2, $3, $4, $5, $6, $7::NUMERIC, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17) ON CONFLICT (\"transactionHash\", \"messageIndex\") DO UPDATE SET \"blockNumber\" = EXCLUDED.\"blockNumber\", \"timestamp\" = EXCLUDED.\"timestamp\"",
			internalTx.BlockNumberNumber, time.Unix(internalTx.TimestampSec, 0).UTC(), internalTx.ThashString, internalTx.SrcString, internalTx.DestString, internalTx.ContractCodeAddrString, internalTx.ValueString, internalTx.Opcode, internalTx.TxType, internalTx.Depth, internalTx.Index, internalTx.InputString, internalTx.CodeString, internalTx.InitialGas, internalTx.LeftOverGas, internalTx.RetString, internalTx.ErrString)
		common.CheckErr(err2, nil)
		rowAffected, err3 := result.RowsAffected()
		common.CheckErr(err3, nil)
		// fmt.Printf("rowsAffected = %d\n", rowsAffected)
		timeNowString := time.Now().UTC().Format("2006-01-02 15:04:05")
		// if rowsAffected == 0 {
		severity := ""
		if rowAffected == 0 {
			severity = "warning"
		}
		fmt.Printf("%s %s: rowAffected == %d, blockNumber = %d, transactionHash = %s, index = %d\n", timeNowString, severity, rowAffected, internalTx.BlockNumberNumber, internalTx.ThashString, internalTx.Index)
		totalRowsAffected += uint64(rowAffected)
	}
	fmt.Printf("len(internalTxStore) = %d, totalRowsAffected = %d\n", len(internalTxStore), totalRowsAffected)
	return totalRowsAffected
}
