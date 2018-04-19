package db

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	_ "github.com/lib/pq"
)

var connStr = "host=" + os.Getenv("DATABASE_HOSTNAME") + " port=" + os.Getenv("DATABASE_PORT") + " dbname=" + os.Getenv("DATABASE_NAME") + " user=" + os.Getenv("DATABASE_USERNAME") + " password=" + os.Getenv("DATABASE_PASSWORD") + " sslmode=disable"
var DBO = connectDB(connStr)

func connectDB(connStr string) *sql.DB {
	dbo, err := sql.Open("postgres", connStr)
	common.CheckErr(err, nil)
	return dbo
}

var FullSyncStartBlock, BlockTime = getFullSyncStartBlock()

func getFullSyncStartBlock() (FullSyncStartBlock uint64, BlockTime uint64) {
	fmt.Printf("flag.Lookup(\"test.v\") = %s\n", flag.Lookup("test.v")) // too strange: if this line is removed, one test will fail!
	if flag.Lookup("test.v") != nil {
		FullSyncStartBlock = 0
		BlockTime = 0
		return
	}
	var dummyBlockNumber uint64
	var epoch1, epoch2 int64
	blockRows, err1 := DBO.Query(`SELECT "blockNumber", EXTRACT(EPOCH FROM MAX(timestamp))::BIGINT FROM internal_message GROUP BY "blockNumber" ORDER BY "blockNumber" DESC LIMIT 2`)
	defer blockRows.Close()
	common.CheckErr(err1, nil)
	if blockRows.Next() {
		err2 := blockRows.Scan(&FullSyncStartBlock, &epoch2)
		common.CheckErr(err2, nil)
	}
	if blockRows.Next() {
		err3 := blockRows.Scan(&dummyBlockNumber, &epoch1)
		common.CheckErr(err3, nil)
	}
	BlockTime = uint64(epoch2 - epoch1)
	fmt.Printf("BlockTime = %d seconds\n", BlockTime)
	FullSyncStartBlockString := os.Getenv("GETH_FULL_SYNC_START_BLOCK")
	if len(FullSyncStartBlockString) > 0 {
		var err error
		FullSyncStartBlock, err = strconv.ParseUint(FullSyncStartBlockString, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("Cannot parse FullSyncStartBlockString: %s", FullSyncStartBlockString))
		}
	}
	fmt.Printf("FullSyncStartBlock = %d\n", FullSyncStartBlock)
	return
}
