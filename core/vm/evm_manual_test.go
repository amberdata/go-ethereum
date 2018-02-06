package vm

import (
  "testing"

	"math/big"

  "fmt"
	"github.com/ethereum/go-ethereum/common"

)
func TestSaveInternalTx(t *testing.T) {
	thash :=  common.HexToHash()
  src := common.HexToAddress()
  dest := common.HexToAddress()SetString('')
  value := big.Int.NewInt(0).
  txType := "CREATE"
  newEVM = &EVM{}
  newEVM.SaveInternalTx(thash, src, dest, value, txType)
  printf("please manually verify in database")
}
