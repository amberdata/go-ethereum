package common

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

var BlockChainId = os.Getenv("BLOCKCHAIN_ID")
var IsInFlux = getIsInFlux()

func getIsInFlux() bool {
	fmt.Printf("flag.Lookup(\"test.v\") = %s\n", flag.Lookup("test.v")) // too strange: if this line is removed, one test will fail!
	if flag.Lookup("test.v") != nil {
		return true
	}
	isInFluxString := os.Getenv("GETH_IS_IN_FLUX")
	if len(isInFluxString) > 0 {
		isInFlux, err := strconv.ParseBool(isInFluxString)
		if err != nil {
			panic(fmt.Sprintf("Cannot parse isInFluxString: %s", isInFluxString))
		}
		fmt.Printf("isInFlux = %t\n", isInFlux)
		return isInFlux
	} else {
		return false
	}
}

var EnableSaveInternalTx = getEnableSaveInternalTx()

func getEnableSaveInternalTx() bool {
	fmt.Printf("flag.Lookup(\"test.v\") = %s\n", flag.Lookup("test.v")) // too strange: if this line is removed, one test will fail!
	if flag.Lookup("test.v") != nil {
		return true
	}
	enableSaveInternalTxString := os.Getenv("GETH_ENABLE_SAVE_INTERNAL_MESSAGE")
	if len(enableSaveInternalTxString) > 0 {
		enableSaveInternalTx, err := strconv.ParseBool(enableSaveInternalTxString)
		if err != nil {
			panic(fmt.Sprintf("Cannot parse enableSaveInternalTxString: %s", enableSaveInternalTxString))
		}
		fmt.Printf("enableSaveInternalTx = %t\n", enableSaveInternalTx)
		return enableSaveInternalTx
	} else {
		return false
	}
}

var EnableSaveInternalTxSql = getEnableSaveInternalTxSql()

func getEnableSaveInternalTxSql() bool {
	fmt.Printf("flag.Lookup(\"test.v\") = %s\n", flag.Lookup("test.v")) // too strange: if this line is removed, one test will fail!
	if flag.Lookup("test.v") != nil {
		return true
	}
	enableSaveInternalTxSqlString := os.Getenv("GETH_ENABLE_SAVE_INTERNAL_MESSAGE_SQL")
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

var EnableSaveInternalTxKafka = getEnableSaveInternalTxKafka()

func getEnableSaveInternalTxKafka() bool {
	fmt.Printf("flag.Lookup(\"test.v\") = %s\n", flag.Lookup("test.v")) // too strange: if this line is removed, one test will fail!
	if flag.Lookup("test.v") != nil {
		return true
	}
	enableSaveInternalTxKafkaString := os.Getenv("GETH_ENABLE_SAVE_INTERNAL_MESSAGE_KAFKA")
	if len(enableSaveInternalTxKafkaString) > 0 {
		enableSaveInternalTxKafka, err := strconv.ParseBool(enableSaveInternalTxKafkaString)
		if err != nil {
			panic(fmt.Sprintf("Cannot parse enableSaveInternalTxKafkaString: %s", enableSaveInternalTxKafkaString))
		}
		fmt.Printf("enableSaveInternalTxKafka = %t\n", enableSaveInternalTxKafka)
		return enableSaveInternalTxKafka
	} else {
		return false
	}
}
