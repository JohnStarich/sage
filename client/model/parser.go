package model

import (
	"github.com/aclindsa/ofxgo"
	"github.com/johnstarich/sage/ledger"
)

type TransactionParser func(*ofxgo.Response) ([]Account, []ledger.Transaction, error)
