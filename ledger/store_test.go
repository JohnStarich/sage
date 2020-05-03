package ledger

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/johnstarich/sage/prompter"
	"github.com/johnstarich/sage/vcs"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func starterStore(t *testing.T) *Store {
	ldg, err := New(nil)
	require.NoError(t, err)
	return &Store{
		Ledger:      ldg,
		logger:      zaptest.NewLogger(t),
		syncing:     atomic.NewBool(false),
		lastSyncErr: atomic.NewError(nil),
		syncFile:    func() error { return nil },
		syncLedger: func(start, end time.Time, download downloader, processTxns txnMutator, ldg *Ledger, logger *zap.Logger, prompt prompter.Prompter) error {
			return nil
		},
	}
}

func TestNewStore(t *testing.T) {
	require.NoError(t, os.Mkdir("repo", 0700))
	defer func() { require.NoError(t, os.RemoveAll("repo")) }()

	repo, err := vcs.Open("repo")
	require.NoError(t, err)
	logger := zaptest.NewLogger(t)
	store, err := NewStore(repo.File("someFile.ledger"), logger)

	require.NoError(t, err)
	assert.NotNil(t, store.Ledger)
	assert.Equal(t, repo.File("someFile.ledger"), store.file)
	assert.Equal(t, logger, store.logger)
	assert.Equal(t, atomic.NewBool(false), store.syncing)
	assert.Equal(t, atomic.NewError(nil), store.lastSyncErr)
	assert.NotNil(t, store.syncFile)
	assert.NotNil(t, store.syncLedger)
}

type mockFile struct {
	buf      bytes.Buffer
	writeErr error
	readErr  error
}

func (m *mockFile) Write(b []byte) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	_, err := m.buf.Write(b)
	return err
}

func (m *mockFile) Read() ([]byte, error) {
	return m.buf.Bytes(), m.readErr
}

func TestNewStoreDeps(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		someFile := &mockFile{}
		store, err := NewStore(someFile, logger)
		require.NoError(t, err)
		assert.NotNil(t, store.Ledger)
		assert.Same(t, someFile, store.file)
		assert.Same(t, logger, store.logger)
		assert.Equal(t, atomic.NewBool(false), store.syncing)
		assert.Equal(t, atomic.NewError(nil), store.lastSyncErr)
		assert.NotNil(t, store.syncFile)
		assert.NotNil(t, store.syncLedger)
	})

	t.Run("error reading file", func(t *testing.T) {
		logger := zaptest.NewLogger(t)
		someFile := &mockFile{readErr: errors.New("some error")}
		_, err := NewStore(someFile, logger)
		require.Error(t, err)
		assert.Equal(t, "Error reading ledger file: some error", err.Error())
	})
}

func TestSync(t *testing.T) {
	for _, tc := range []struct {
		description   string
		syncLedgerErr error
		syncFileErr   error
		expectSyncErr string
	}{
		{
			description: "happy path",
		},
		{
			description:   "ledger sync error",
			syncLedgerErr: errors.New("some error"),
			expectSyncErr: "some error",
		},
		{
			description:   "ledger sync validation error",
			syncLedgerErr: NewValidateError(0, errors.New("some error")),
			expectSyncErr: "Failed to validate ledger at transaction index #0: some error",
		},
		{
			description:   "file sync error",
			syncFileErr:   errors.New("some error"),
			expectSyncErr: "Error writing ledger to disk: some error",
		},
		{
			description:   "ledger sync validation error hidden if file sync error",
			syncLedgerErr: NewValidateError(0, errors.New("some validation error")),
			syncFileErr:   errors.New("some file error"),
			expectSyncErr: "Error writing ledger to disk: some file error",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			inputStart, inputEnd := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC), time.Date(2020, time.January, 2, 0, 0, 0, 0, time.UTC)
			someTxns := []Transaction{{}, {}}
			store := starterStore(t)
			store.syncLedger = func(start, end time.Time, download downloader, processTxns txnMutator, ldg *Ledger, logger *zap.Logger, prompt prompter.Prompter) error {
				assert.Equal(t, inputStart, start)
				assert.Equal(t, inputEnd, end)
				// run funcs to assert they're the right ones
				_, _ = download(start, end, prompt)
				processTxns(someTxns)
				return tc.syncLedgerErr
			}
			store.syncFile = func() error {
				return tc.syncFileErr
			}
			ranDownload := false
			download := func(start, end time.Time, prompt prompter.Prompter) ([]Transaction, error) {
				ranDownload = true
				assert.Equal(t, inputStart, start)
				assert.Equal(t, inputEnd, end)
				return nil, nil
			}
			ranProcessTxns := false
			processTxns := func(txns []Transaction) {
				ranProcessTxns = true
				assert.Equal(t, someTxns, txns)
			}

			store.StartSync(inputStart, inputEnd, download, processTxns)
			var syncing bool
			var syncErr error
			const (
				sleep = 100 * time.Millisecond
				wait  = time.Second
			)
			for i := 0; i < int(wait/sleep); i++ {
				// TODO check prompt request
				syncing, _, syncErr = store.SyncStatus()
				if !syncing {
					break
				}
				time.Sleep(sleep)
			}
			assert.False(t, syncing, "syncing should be complete")
			if tc.expectSyncErr != "" {
				require.Error(t, syncErr)
				assert.Equal(t, tc.expectSyncErr, syncErr.Error())
			} else {
				assert.NoError(t, syncErr)
			}
			assert.True(t, ranDownload, "Download func should be called")
			assert.True(t, ranProcessTxns, "ProcessTxns func should be called")
		})
	}
}

func TestSyncMutex(t *testing.T) {
	// syncing many times concurrently should not execute more than once
	syncCount := atomic.NewInt32(0)
	wait := make(chan bool)
	store := starterStore(t)
	store.syncLedger = func(start, end time.Time, download downloader, processTxns txnMutator, ldg *Ledger, logger *zap.Logger, prompt prompter.Prompter) error {
		syncCount.Inc()
		<-wait
		return nil
	}
	var someTime time.Time
	for i := 0; i < 100; i++ {
		store.StartSync(someTime, someTime, func(start, end time.Time, prompt prompter.Prompter) ([]Transaction, error) { return nil, nil }, func([]Transaction) {})
	}
	wait <- true
	assert.EqualValues(t, 1, syncCount.Load())
}

func TestSyncLedgerFile(t *testing.T) {
	ldg, err := New(nil)
	require.NoError(t, err)

	t.Run("successful write", func(t *testing.T) {
		file := &mockFile{}
		syncFile := syncLedgerFile(ldg, file)
		assert.NoError(t, syncFile())
		assert.Equal(t, "", file.buf.String())
	})

	t.Run("failed write", func(t *testing.T) {
		file := &mockFile{writeErr: errors.New("some error")}
		syncFile := syncLedgerFile(ldg, file)
		err := syncFile()
		require.Error(t, err)
		assert.Equal(t, "Error writing ledger to disk: some error", err.Error())
	})
}

func TestSyncLedger(t *testing.T) {
	someTxn := func(date string) Transaction {
		return Transaction{
			Date:  parseDate(t, date),
			Payee: "some store",
			Postings: []Posting{
				{Account: "assets", Amount: *decFloat(10)},
				{Account: "expenses", Amount: *decFloat(-10)},
			},
		}
	}
	someTxnPtr := func(date string) *Transaction {
		txn := someTxn(date)
		return &txn
	}
	for _, tc := range []struct {
		description       string
		start, end        time.Time
		initialTxns       []Transaction
		downloadTxns      [][]Transaction
		downloadErr       []error
		downloadTimes     []time.Time // {start, end, start, end, ...}
		expectTxns        Transactions
		expectProcessTxns bool
		expectErr         string
	}{
		{
			description: "invalid ledger",
			initialTxns: []Transaction{{}},
			expectErr:   "Existing ledger is not valid",
			expectTxns:  Transactions{{}},
		},
		{
			description:       "sync 1 day",
			start:             parseDate(t, "2020/01/05"),
			end:               parseDate(t, "2020/01/06"),
			downloadTxns:      [][]Transaction{{someTxn("2020/01/05")}},
			downloadTimes:     []time.Time{parseDate(t, "2020/01/03"), parseDate(t, "2020/01/06")},
			expectTxns:        Transactions{someTxnPtr("2020/01/05")},
			expectProcessTxns: true,
		},
		{
			description: "sync 3.5 months",
			start:       parseDate(t, "2020/01/03"),
			end:         parseDate(t, "2020/04/15"),
			downloadTxns: [][]Transaction{
				{someTxn("2020/01/10")},
				{someTxn("2020/02/10")},
				{someTxn("2020/03/10")},
				{someTxn("2020/04/10")},
			},
			downloadTimes: []time.Time{
				parseDate(t, "2020/01/01"), parseDate(t, "2020/01/31"),
				parseDate(t, "2020/01/31"), parseDate(t, "2020/03/01"),
				parseDate(t, "2020/03/01"), parseDate(t, "2020/03/31"),
				parseDate(t, "2020/03/31"), parseDate(t, "2020/04/15"),
			},
			expectTxns: Transactions{
				someTxnPtr("2020/01/10"),
				someTxnPtr("2020/02/10"),
				someTxnPtr("2020/03/10"),
				someTxnPtr("2020/04/10"),
			},
			expectProcessTxns: true,
		},
		{
			description:       "invalid downloaded txn",
			start:             parseDate(t, "2020/01/03"),
			end:               parseDate(t, "2020/01/03"),
			downloadErr:       []error{errors.New("some error")},
			downloadTimes:     []time.Time{parseDate(t, "2020/01/01"), parseDate(t, "2020/01/03")},
			expectErr:         "some error",
			expectProcessTxns: true,
		},
		{
			description: "invalid downloaded txn multiple months",
			start:       parseDate(t, "2020/01/03"),
			end:         parseDate(t, "2020/02/29"),
			downloadTxns: [][]Transaction{
				nil,
				{someTxn("2020/02/15")},
			},
			downloadErr: []error{
				errors.New("some error"),
				nil,
			},
			downloadTimes: []time.Time{
				parseDate(t, "2020/01/01"), parseDate(t, "2020/01/31"),
				parseDate(t, "2020/01/31"), parseDate(t, "2020/02/29"),
			},
			expectTxns:        Transactions{someTxnPtr("2020/02/15")},
			expectErr:         "some error",
			expectProcessTxns: true,
		},
		{
			description:       "filter out txns before start",
			start:             parseDate(t, "2020/01/03"),
			end:               parseDate(t, "2020/01/03"),
			downloadTxns:      [][]Transaction{{someTxn("2020/01/01")}},
			downloadTimes:     []time.Time{parseDate(t, "2020/01/01"), parseDate(t, "2020/01/03")},
			expectTxns:        nil,
			expectProcessTxns: true,
		},
		{
			description:       "error adding transaction",
			start:             parseDate(t, "2020/01/03"),
			end:               parseDate(t, "2020/01/03"),
			downloadTxns:      [][]Transaction{{{Date: parseDate(t, "2020/01/03")}}}, // invalid txn
			downloadTimes:     []time.Time{parseDate(t, "2020/01/01"), parseDate(t, "2020/01/03")},
			expectTxns:        Transactions{},
			expectProcessTxns: true,
			expectErr:         "Failed to validate ledger",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			downloadIndex := 0
			download := func(start, end time.Time, prompt prompter.Prompter) (txns []Transaction, err error) {
				require.Less(t, downloadIndex*2+1, len(tc.downloadTimes), "Not enough start/end pairs in test case")
				assert.Equal(t, tc.downloadTimes[downloadIndex*2], start, "Start date for download index %d should be %s", downloadIndex, tc.downloadTimes[downloadIndex*2])
				assert.Equal(t, tc.downloadTimes[downloadIndex*2+1], end, "End date for download index %d should be %s", downloadIndex, tc.downloadTimes[downloadIndex*2+1])
				if downloadIndex < len(tc.downloadTxns) {
					txns = tc.downloadTxns[downloadIndex]
				}
				if downloadIndex < len(tc.downloadErr) {
					err = tc.downloadErr[downloadIndex]
				}
				downloadIndex++
				return txns, err
			}
			ranProcessTxns := false
			processTxns := func(txns []Transaction) {
				ranProcessTxns = true
				assert.Equal(t, afterDate(tc.start, flatten(tc.downloadTxns...)), txns)
			}
			ldg, err := New(tc.initialTxns)
			require.NoError(t, err)

			err = syncLedger(tc.start, tc.end, download, processTxns, ldg, logger, prompter.New())
			assert.Equal(t, tc.expectTxns, ldg.transactions)
			assert.Equal(t, tc.expectProcessTxns, ranProcessTxns, "Process txns did not match expectation")
			if tc.expectErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, len(tc.downloadTimes)/2, downloadIndex)
		})
	}
}

func flatten(txns ...[]Transaction) []Transaction {
	flattened := make([]Transaction, 0)
	for _, txnSlice := range txns {
		flattened = append(flattened, txnSlice...)
	}
	return flattened
}

func afterDate(start time.Time, txns []Transaction) []Transaction {
	result := make([]Transaction, 0)
	for _, txn := range txns {
		if !txn.Date.Before(start) {
			result = append(result, txn)
		}
	}
	return result
}

func TestSyncRecent(t *testing.T) {
	someTxn := func(date string) Transaction {
		return Transaction{
			Date:  parseDate(t, date),
			Payee: "some store",
			Postings: []Posting{
				{Account: "assets", Amount: *decFloat(10)},
				{Account: "expenses", Amount: *decFloat(-10)},
			},
		}
	}
	formatDate := func(date time.Time) string {
		return date.Format("2006/01/02")
	}

	for _, tc := range []struct {
		description            string
		txns                   []Transaction
		expectStart, expectEnd string
	}{
		{
			description: "some txns",
			txns:        []Transaction{someTxn("2020/01/01"), someTxn("2020/01/02")},
			expectStart: "2020/01/02",
			expectEnd:   formatDate(currentDate()),
		},
		{
			description: "no txns",
			txns:        []Transaction{},
			expectStart: formatDate(currentDate().Add(-30 * day)),
			expectEnd:   formatDate(currentDate()),
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			var ranDownload, ranProcess atomic.Bool
			download := func(start, end time.Time, prompt prompter.Prompter) ([]Transaction, error) {
				ranDownload.Store(true)
				return nil, nil
			}
			processTxns := func([]Transaction) {
				ranProcess.Store(true)
			}
			ldg, err := New(tc.txns)
			require.NoError(t, err)
			wait := make(chan bool)
			store := starterStore(t)
			store.Ledger = ldg
			store.syncLedger = func(start, end time.Time, download downloader, processTxns txnMutator, ldg *Ledger, logger *zap.Logger, prompt prompter.Prompter) error {
				assert.Equal(t, parseDate(t, tc.expectStart), start, "Start date is incorrect")
				assert.Equal(t, parseDate(t, tc.expectEnd), end, "End date is incorrect")
				_, _ = download(start, end, prompt)
				processTxns(nil)
				wait <- true
				return nil
			}
			store.SyncRecent(download, processTxns)
			<-wait
			assert.True(t, ranDownload.Load())
			assert.True(t, ranProcess.Load())
		})
	}
}

func TestResync(t *testing.T) {
	someTxn := func(date string) Transaction {
		return Transaction{
			Date:  parseDate(t, date),
			Payee: "some store",
			Postings: []Posting{
				{Account: "assets", Amount: *decFloat(10)},
				{Account: "expenses", Amount: *decFloat(-10)},
			},
		}
	}

	var ranDownload, ranProcess atomic.Bool
	download := func(start, end time.Time, prompt prompter.Prompter) ([]Transaction, error) {
		ranDownload.Store(true)
		return nil, nil
	}
	processTxns := func([]Transaction) {
		ranProcess.Store(true)
	}
	ldg, err := New([]Transaction{
		someTxn("2020/01/01"),
		someTxn("2020/01/02"),
	})
	require.NoError(t, err)
	wait := make(chan bool)
	store := starterStore(t)
	store.Ledger = ldg
	store.syncLedger = func(start, end time.Time, download downloader, processTxns txnMutator, ldg *Ledger, logger *zap.Logger, prompt prompter.Prompter) error {
		assert.Equal(t, parseDate(t, "2020/01/01"), start, "Start date is incorrect")
		assert.Equal(t, currentDate(), end, "End date is incorrect")
		_, _ = download(start, end, prompt)
		processTxns(nil)
		wait <- true
		return errors.New("stop early")
	}
	store.Resync(download, processTxns)
	<-wait
	assert.True(t, ranDownload.Load())
	assert.True(t, ranProcess.Load())
}

func TestStoreRenameAccount(t *testing.T) {
	ranSync := false
	syncFile := func() error {
		ranSync = true
		return nil
	}
	store := starterStore(t)
	store.syncFile = syncFile
	_, _ = store.RenameAccount("", "", "", "")
	assert.True(t, ranSync)
}

func TestStoreUpdateAccount(t *testing.T) {
	ranSync := false
	syncFile := func() error {
		ranSync = true
		return nil
	}
	store := starterStore(t)
	store.syncFile = syncFile
	_ = store.UpdateAccount("x", "x")
	assert.True(t, ranSync)
}

func TestStoreAddTransactions(t *testing.T) {
	ranSync := false
	syncFile := func() error {
		ranSync = true
		return nil
	}
	store := starterStore(t)
	store.syncFile = syncFile
	_ = store.AddTransactions(nil)
	assert.True(t, ranSync)
}

func TestStoreUpdateTransaction(t *testing.T) {
	txn := Transaction{
		Payee: "some payee",
		Postings: []Posting{
			{Account: "assets", Tags: map[string]string{idTag: "my-txn"}},
			{Account: "expenses"},
		},
	}
	ldg, err := New([]Transaction{txn})
	require.NoError(t, err)
	ranSync := false
	syncFile := func() error {
		ranSync = true
		return nil
	}
	store := starterStore(t)
	store.Ledger = ldg
	store.syncFile = syncFile
	_ = store.UpdateTransaction("my-txn", txn)
	assert.True(t, ranSync)
}

func TestUpdateTransactions(t *testing.T) {
	txn1 := Transaction{Payee: "some payee", Postings: []Posting{
		{Account: "assets", Amount: *decFloat(10), Tags: map[string]string{idTag: "txn1"}},
		{Account: "expenses", Amount: *decFloat(-10)},
	}}
	txn2 := Transaction{Payee: "some payee", Postings: []Posting{
		{Account: "assets", Amount: *decFloat(10), Tags: map[string]string{idTag: "txn2"}},
		{Account: "expenses", Amount: *decFloat(-10)},
	}}
	for _, tc := range []struct {
		description string
		txns        map[string]Transaction
		expectSync  bool
		expectErr   bool
	}{
		{
			description: "happy path",
			txns: map[string]Transaction{
				"txn1": txn1,
				"txn2": txn2,
			},
			expectSync: true,
		},
		{
			description: "non-critical ledger error",
			txns: map[string]Transaction{
				"txn1": {Postings: []Posting{{Account: "something"}}},
			},
			expectSync: true,
			expectErr:  true,
		},
		{
			description: "critical error",
			txns: map[string]Transaction{
				"txn100": {},
			},
			expectSync: false,
			expectErr:  true,
		},
		{
			description: "critical error takes priority",
			txns: map[string]Transaction{
				"txn1":   {Postings: []Posting{{Account: "something"}}},
				"txn100": {},
			},
			expectSync: false,
			expectErr:  true,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			ldg, err := New([]Transaction{txn1, txn2})
			require.NoError(t, err)
			ranSync := false
			syncFile := func() error {
				ranSync = true
				return nil
			}
			store := starterStore(t)
			store.Ledger = ldg
			store.syncFile = syncFile
			err = store.UpdateTransactions(tc.txns)
			assert.Equal(t, tc.expectSync, ranSync, "Sync run didn't match expectation")
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestStoreUpdateOpeningBalance(t *testing.T) {
	ranSync := false
	syncFile := func() error {
		ranSync = true
		return nil
	}
	store := starterStore(t)
	store.syncFile = syncFile
	err := store.UpdateOpeningBalance(Transaction{
		Date: parseDate(t, "2020/01/01"),
		Postings: []Posting{
			{Account: "assets", Amount: *decFloat(10)},
			{Account: "expenses", Amount: *decFloat(-10), Tags: map[string]string{idTag: OpeningBalanceID}},
		},
	})
	assert.NoError(t, err)
	assert.True(t, ranSync)
}
