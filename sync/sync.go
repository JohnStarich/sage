package sync

import (
	"fmt"
	"os"

	"github.com/johnstarich/sage/plaindb"
)

func Shutdown(db plaindb.DB, exitCode int) {
	fmt.Println(`{"level":"info","msg":"Shutting down"}`)
	_ = db.Close()
	ledgerMu.Lock()
	rulesMu.Lock()
	os.Exit(exitCode)
}
