// Copyright 2014 The go-ethereum Authors
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

package vm

import (
	"math/big"
	"strings"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	_ "github.com/lib/pq"
)

// emptyCodeHash is used by create to ensure deployment is disallowed to already
// deployed contract addresses (relevant after the account abstraction).
var emptyCodeHash = crypto.Keccak256Hash(nil)
var internalTxTypeMap = map[string]int{
	"unknown":                       0,
	"contract_creation_by_contract": 1,
	"contract_creation_by_eoa":      2,
	"c2c": 3,
	"c2e": 4,
	"e2c": 5,
	"e2e": 6,
}

// var enableSaveInternalTx = getEnableSaveInternalTx()
//
// func getEnableSaveInternalTx() bool {
// 	fmt.Printf("flag.Lookup(\"test.v\") = %s\n", flag.Lookup("test.v")) // too strange: if this line is removed, one test will fail!
// 	if flag.Lookup("test.v") != nil {
// 		return true
// 	}
// 	enableSaveInternalTxString := os.Getenv("GETH_ENABLE_SAVE_INTERNAL_MESSAGE")
// 	if len(enableSaveInternalTxString) > 0 {
// 		enableSaveInternalTx, err := strconv.ParseBool(enableSaveInternalTxString)
// 		if err != nil {
// 			panic(fmt.Sprintf("Cannot parse enableSaveInternalTxString: %s", enableSaveInternalTxString))
// 		}
// 		fmt.Printf("enableSaveInternalTx = %t\n", enableSaveInternalTx)
// 		return enableSaveInternalTx
// 	} else {
// 		return false
// 	}
// }

type (
	CanTransferFunc func(StateDB, common.Address, *big.Int) bool
	TransferFunc    func(StateDB, common.Address, common.Address, *big.Int)
	// GetHashFunc returns the nth block hash in the blockchain
	// and is used by the BLOCKHASH EVM op code.
	GetHashFunc func(uint64) common.Hash
)

// run runs the given contract and takes care of running precompiles with a fallback to the byte code interpreter.
func run(evm *EVM, snapshot int, contract *Contract, input []byte) ([]byte, error) {
	if contract.CodeAddr != nil {
		precompiles := PrecompiledContractsHomestead
		if evm.ChainConfig().IsByzantium(evm.BlockNumber) {
			precompiles = PrecompiledContractsByzantium
		}
		if p := precompiles[*contract.CodeAddr]; p != nil {
			return RunPrecompiledContract(p, input, contract)
		}
	}
	return evm.interpreter.Run(snapshot, contract, input)
}

// Context provides the EVM with auxiliary information. Once provided
// it shouldn't be modified.
type Context struct {
	// CanTransfer returns whether the account contains
	// sufficient ether to transfer the value
	CanTransfer CanTransferFunc
	// Transfer transfers ether from one account to the other
	Transfer TransferFunc
	// GetHash returns the hash corresponding to n
	GetHash GetHashFunc

	// Message information
	Origin   common.Address // Provides information for ORIGIN
	GasPrice *big.Int       // Provides information for GASPRICE

	// Block information
	Coinbase    common.Address // Provides information for COINBASE
	GasLimit    *big.Int       // Provides information for GASLIMIT
	BlockNumber *big.Int       // Provides information for NUMBER
	Time        *big.Int       // Provides information for TIME
	Difficulty  *big.Int       // Provides information for DIFFICULTY
}

// EVM is the Ethereum Virtual Machine base object and provides
// the necessary tools to run a contract on the given state with
// the provided context. It should be noted that any error
// generated through any of the calls should be considered a
// revert-state-and-consume-all-gas operation, no checks on
// specific errors should ever be performed. The interpreter makes
// sure that any errors generated are to be considered faulty code.
//
// The EVM should never be reused and is not thread safe.
type EVM struct {
	// Context provides auxiliary blockchain related information
	Context
	// StateDB gives access to the underlying state
	StateDB StateDB
	// Depth is the current call stack
	depth int

	// chainConfig contains information about the current chain
	chainConfig *params.ChainConfig
	// chain rules contains the chain rules for the current epoch
	chainRules params.Rules
	// virtual machine configuration options used to initialise the
	// evm.
	vmConfig Config
	// global (to this context) ethereum virtual machine
	// used throughout the execution of the tx.
	interpreter *Interpreter
	// abort is used to abort the EVM calling operations
	// NOTE: must be set atomically
	abort int32

	InternalTxIndex        uint64
	InternalTxStore        []*types.InternalTx
	createdContractAddrMap map[string]bool
}

// NewEVM retutrns a new EVM . The returned EVM is not thread safe and should
// only ever be used *once*.
func NewEVM(ctx Context, statedb StateDB, chainConfig *params.ChainConfig, vmConfig Config) *EVM {
	evm := &EVM{
		Context:                ctx,
		StateDB:                statedb,
		vmConfig:               vmConfig,
		chainConfig:            chainConfig,
		chainRules:             chainConfig.Rules(ctx.BlockNumber),
		InternalTxIndex:        0,
		InternalTxStore:        []*types.InternalTx{},
		createdContractAddrMap: map[string]bool{},
	}

	evm.interpreter = NewInterpreter(evm, vmConfig)
	return evm
}

// Cancel cancels any running EVM operation. This may be called concurrently and
// it's safe to be called multiple times.
func (evm *EVM) Cancel() {
	atomic.StoreInt32(&evm.abort, 1)
}

// Call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (evm *EVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !evm.Context.CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, gas, ErrInsufficientBalance
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.StateDB.Snapshot()
	)
	if !evm.StateDB.Exist(addr) {
		precompiles := PrecompiledContractsHomestead
		if evm.ChainConfig().IsByzantium(evm.BlockNumber) {
			precompiles = PrecompiledContractsByzantium
		}
		if precompiles[addr] == nil && evm.ChainConfig().IsEIP158(evm.BlockNumber) && value.Sign() == 0 {
			return nil, gas, nil
		}
		evm.StateDB.CreateAccount(addr)
	}
	evm.Transfer(evm.StateDB, caller.Address(), to.Address(), value)

	// initialise a new contract and set the code that is to be used by the
	// E The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))
	// fmt.Printf("before run: %d\n", contract.Gas)
	ret, err = run(evm, snapshot, contract, input)
	// fmt.Printf("after run: %d\n", contract.Gas)
	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	// fmt.Printf("emptyCodeHash = %d\n", emptyCodeHash)
	// fmt.Printf("evm depth in call is %d\n", evm.depth)
	// fmt.Printf("evm.StateDB.GetCodeHash(to.Address()) = %d\n", evm.StateDB.GetCodeHash(to.Address()))

	// evm.CheckInvariant(contract.Caller())
	txType := evm.address2internalTxType(evm.depth, to.Address())
	if txType != internalTxTypeMap["e2e"] {
		evm.SaveInternalTx(evm.BlockNumber, evm.Time, evm.StateDB.(*state.StateDB).GetThash(), contract.Caller(), to.Address(), addr, value, "CALL", txType, evm.depth, evm.InternalTxIndex, input, nil, gas, contract.Gas, ret, err)
	}

	return ret, contract.Gas, err
}

// CallCode executes the contract associated with the addr with the given input
// as parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
//
// CallCode differs from Call in the sense that it executes the given address'
// code with the caller as context.
func (evm *EVM) CallCode(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !evm.CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, gas, ErrInsufficientBalance
	}

	var (
		snapshot = evm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)
	// initialise a new contract and set the code that is to be used by the
	// E The contract is a scoped evmironment for this execution context
	// only.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	ret, err = run(evm, snapshot, contract, input)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}

	// evm.CheckInvariant(contract.Caller())
	evm.SaveInternalTx(evm.BlockNumber, evm.Time, evm.StateDB.(*state.StateDB).GetThash(), contract.Caller(), to.Address(), addr, value, "CALLCODE", internalTxTypeMap["c2c"], evm.depth, evm.InternalTxIndex, input, nil, gas, contract.Gas, ret, err)

	return ret, contract.Gas, err
}

// DelegateCall executes the contract associated with the addr with the given input
// as parameters. It reverses the state in case of an execution error.
//
// DelegateCall differs from CallCode in the sense that it executes the given address'
// code with the caller as context and the caller is set to the caller of the caller.
func (evm *EVM) DelegateCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	var (
		snapshot = evm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)

	// Initialise a new contract and make initialise the delegate values
	contract := NewContract(caller, to, nil, gas).AsDelegate()
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	ret, err = run(evm, snapshot, contract, input)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}

	// evm.CheckInvariant(contract.Caller())
	txType := internalTxTypeMap["c2c"]
	if evm.depth == 1 {
		txType = internalTxTypeMap["e2c"]
	}
	evm.SaveInternalTx(evm.BlockNumber, evm.Time, evm.StateDB.(*state.StateDB).GetThash(), contract.Caller(), to.Address(), addr, nil, "DELEGATECALL", txType, evm.depth, evm.InternalTxIndex, input, nil, gas, contract.Gas, ret, err)

	return ret, contract.Gas, err
}

// StaticCall executes the contract associated with the addr with the given input
// as parameters while disallowing any modifications to the state during the call.
// Opcodes that attempt to perform such modifications will result in exceptions
// instead of performing the modifications.
func (evm *EVM) StaticCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Make sure the readonly is only set if we aren't in readonly yet
	// this makes also sure that the readonly flag isn't removed for
	// child calls.
	if !evm.interpreter.readOnly {
		evm.interpreter.readOnly = true
		defer func() { evm.interpreter.readOnly = false }()
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.StateDB.Snapshot()
	)
	// Initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, new(big.Int), gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in Homestead this also counts for code storage gas errors.
	ret, err = run(evm, snapshot, contract, input)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// Create creates a new contract using code as deployment code.
func (evm *EVM) Create(caller ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {

	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if evm.depth > int(params.CallCreateDepth) {
		return nil, common.Address{}, gas, ErrDepth
	}
	if !evm.CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, common.Address{}, gas, ErrInsufficientBalance
	}
	// Ensure there's no existing contract already at the designated address
	nonce := evm.StateDB.GetNonce(caller.Address())
	evm.StateDB.SetNonce(caller.Address(), nonce+1)

	contractAddr = crypto.CreateAddress(caller.Address(), nonce)
	contractHash := evm.StateDB.GetCodeHash(contractAddr)
	if evm.StateDB.GetNonce(contractAddr) != 0 || (contractHash != (common.Hash{}) && contractHash != emptyCodeHash) {
		return nil, common.Address{}, 0, ErrContractAddressCollision
	}
	// Create a new account on the state
	snapshot := evm.StateDB.Snapshot()
	evm.StateDB.CreateAccount(contractAddr)
	if evm.ChainConfig().IsEIP158(evm.BlockNumber) {
		evm.StateDB.SetNonce(contractAddr, 1)
	}
	evm.Transfer(evm.StateDB, caller.Address(), contractAddr, value)

	// initialise a new contract and set the code that is to be used by the
	// E The contract is a scoped evmironment for this execution context
	// only.
	contract := NewContract(caller, AccountRef(contractAddr), value, gas)
	contract.SetCallCode(&contractAddr, crypto.Keccak256Hash(code), code)

	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, contractAddr, gas, nil
	}
	evm.createdContractAddrMap[strings.ToLower(contractAddr.Hex())] = true
	ret, err = run(evm, snapshot, contract, nil)
	// check whether the max code size has been exceeded
	maxCodeSizeExceeded := evm.ChainConfig().IsEIP158(evm.BlockNumber) && len(ret) > params.MaxCodeSize
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && !maxCodeSizeExceeded {
		createDataGas := uint64(len(ret)) * params.CreateDataGas
		if contract.UseGas(createDataGas) {
			evm.StateDB.SetCode(contractAddr, ret)
		} else {
			err = ErrCodeStoreOutOfGas
		}
	}

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded || (err != nil && (evm.ChainConfig().IsHomestead(evm.BlockNumber) || err != ErrCodeStoreOutOfGas)) {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	// Assign err if contract code size exceeds the max while the err is still empty.
	if maxCodeSizeExceeded && err == nil {
		err = errMaxCodeSizeExceeded
	}

	// fmt.Printf("evm depth in create is %d\n", evm.depth)
	// fmt.Printf("len(evm.StateDB.GetCodeHash(caller.Address())) = %d\n", len(evm.StateDB.GetCodeHash(caller.Address())))

	// evm.CheckInvariant(contract.Caller())
	txType := internalTxTypeMap["contract_creation_by_contract"]
	if evm.depth == 0 {
		txType = internalTxTypeMap["contract_creation_by_eoa"]
	}
	evm.SaveInternalTx(evm.BlockNumber, evm.Time, evm.StateDB.(*state.StateDB).GetThash(), contract.Caller(), contractAddr, contractAddr, value, "CREATE", txType, evm.depth, evm.InternalTxIndex, nil, code, gas, contract.Gas, ret, err)

	return ret, contractAddr, contract.Gas, err
}

// ChainConfig returns the evmironment's chain configuration
func (evm *EVM) ChainConfig() *params.ChainConfig { return evm.chainConfig }

// Interpreter returns the EVM interpreter
func (evm *EVM) Interpreter() *Interpreter { return evm.interpreter }

func (evm *EVM) SaveInternalTx(blockNumber *big.Int, timestamp *big.Int, thash common.Hash, src common.Address, dest common.Address, contractCodeAddr common.Address, value *big.Int, opcode string, txType int, depth int, index uint64, input []byte, code []byte, initialGas uint64, leftOverGas uint64, ret []byte, err error) uint64 {
	if !common.EnableSaveInternalTx {
		return 0
	}
	valueBigInt := big.NewInt(0)
	if value != nil {
		valueBigInt = value
	}
	inputString := ""
	if len(input) > 0 {
		inputString = hexutil.Encode(input)
	}
	codeString := ""
	if len(code) > 0 {
		codeString = hexutil.Encode(code)
	}
	retString := ""
	if len(ret) > 0 {
		retString = hexutil.Encode(ret)
	}
	errString := ""
	if err != nil {
		errString = err.Error()
	}
	evm.InternalTxStore = append(evm.InternalTxStore, &types.InternalTx{
		BlockNumberNumber:      blockNumber.Uint64(),
		TimestampSec:           timestamp.Int64(),
		ThashString:            strings.ToLower(thash.Hex()),
		SrcString:              strings.ToLower(src.Hex()),
		DestString:             strings.ToLower(dest.Hex()),
		ContractCodeAddrString: strings.ToLower(contractCodeAddr.Hex()),
		ValueBigInt:            valueBigInt,
		Opcode:                 opcode,
		TxType:                 txType,
		Depth:                  depth,
		Index:                  index,
		InputString:            inputString,
		CodeString:             codeString,
		InitialGas:             initialGas,
		LeftOverGas:            leftOverGas,
		RetString:              retString,
		ErrString:              errString,
	})
	evm.InternalTxIndex++
	return 1
}

func (evm *EVM) address2internalTxType(depth int, dest common.Address) int {
	// startTimestamp := time.Now().UTC()
	contractHash := evm.StateDB.GetCodeHash(dest)
	// endTimestamp := time.Now().UTC()
	// elapsed := endTimestamp.Sub(startTimestamp)
	// fmt.Printf("evm.StateDB.GetCodeHash took %d nanosecond\n", elapsed)
	if (contractHash == emptyCodeHash || contractHash == common.Hash{}) {
		if depth > 0 {
			if evm.createdContractAddrMap[strings.ToLower(dest.Hex())] {
				return internalTxTypeMap["c2c"]
			} else {
				return internalTxTypeMap["c2e"]
			}
		} else {
			return internalTxTypeMap["e2e"]
		}
	} else {
		if depth > 0 {
			return internalTxTypeMap["c2c"]
		} else {
			return internalTxTypeMap["e2c"]
		}
	}
}

// func (evm *EVM) CheckInvariant(src common.Address) {
// 	contractHash := evm.StateDB.GetCodeHash(src)
// 	if evm.depth > 0 && (contractHash == common.Hash{} || contractHash == emptyCodeHash) {
// 		panic(fmt.Sprintf("evm depth is %d, contractHash = %d\n", evm.depth, contractHash))
// 	}
// }

func (evm *EVM) GetDepth() int {
	return evm.depth
}
