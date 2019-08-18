package sync

import (
	"fmt"
	"os"
)

func Shutdown(exitCode int) {
	fmt.Println(`{"level":"info","msg":"Shutting down"}`)
	accountsMu.Lock()
	ledgerMu.Lock()
	rulesMu.Lock()
	os.Exit(exitCode)
}
