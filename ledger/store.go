package ledger

import (
	"bytes"
	"io/ioutil"
	"time"

	sErrors "github.com/johnstarich/sage/errors"
	"github.com/johnstarich/sage/pipe"
	"github.com/johnstarich/sage/vcs"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

const (
	day = 24 * time.Hour
)

// Store enables ledger syncing both in memory and on disk
type Store struct {
	*Ledger
	file   vcs.File
	logger *zap.Logger

	syncing     *atomic.Bool
	lastSyncErr *atomic.Error

	syncFile   func() error
	syncLedger func(start, end time.Time, download downloader, processTxns txnMutator, ldg *Ledger, logger *zap.Logger) error
}

// NewStore creates a Ledger Store from the given file
func NewStore(file vcs.File, logger *zap.Logger) (*Store, error) {
	ledgerBytes, err := file.Read()
	if err != nil {
		return nil, errors.Wrap(err, "Error reading ledger file")
	}
	r := ioutil.NopCloser(bytes.NewBuffer(ledgerBytes))

	ldg, err := NewFromReader(r)
	return &Store{
		Ledger:      ldg,
		file:        file,
		logger:      logger,
		syncing:     atomic.NewBool(false),
		lastSyncErr: atomic.NewError(nil),
		syncFile:    syncLedgerFile(ldg, file),
		syncLedger:  syncLedger,
	}, err
}

type downloader func(start, end time.Time) ([]Transaction, error)

type txnMutator func(txns []Transaction)

// StartSync asynchronously downloads and processes new transactions between the start and end dates
// If a partial failure occurs during the sync, writes to disk anyway
func (s *Store) StartSync(start, end time.Time, download downloader, processTxns txnMutator) {
	if !s.startSync() {
		// sync already running
		return
	}
	go func() {
		err := s.sync(start, end, download, processTxns)
		s.stopSync(err)
	}()
}

func (s *Store) sync(start, end time.Time, download downloader, processTxns txnMutator) error {
	ledgerErr := s.syncLedger(start, end, download, processTxns, s.Ledger, s.logger)
	if _, ok := ledgerErr.(Error); ledgerErr != nil && !ok {
		return ledgerErr
	}

	if fileErr := s.syncFile(); fileErr != nil {
		return errors.Wrap(fileErr, "Error writing ledger to disk")
	}
	// save partial errors only if there isn't a more important failure
	return ledgerErr
}

func syncLedgerFile(ldg *Ledger, file vcs.File) func() error {
	return func() error {
		err := file.Write([]byte(ldg.String()))
		return errors.Wrap(err, "Error writing ledger to disk")
	}
}

func syncLedger(start, end time.Time, download downloader, processTxns txnMutator, ldg *Ledger, logger *zap.Logger) error {
	if err := ldg.Validate(); err != nil {
		return errors.Wrap(err, "Existing ledger is not valid")
	}

	const syncBuffer = 2 * day
	duration := end.Sub(start)
	duration += syncBuffer

	const maxDownloadDuration = 30 * day // TODO move 30 day chunking logic into downloader?

	var allTxns []Transaction
	downloadStart := end.Add(-duration)
	var errs sErrors.Errors
	for downloadStart.Before(end) {
		downloadEnd := min(end, downloadStart.Add(maxDownloadDuration))
		logger.Info("Downloading txns...", zap.Time("start", downloadStart), zap.Time("end", downloadEnd))
		txns, err := download(downloadStart, downloadEnd)
		errs.AddErr(err)
		allTxns = append(allTxns, txns...)
		downloadStart = downloadEnd
	}
	if len(errs) == 0 {
		logger.Info("Download succeeded!")
	} else {
		logger.Warn("Failed to download some transactions", zap.Error(errs))
	}

	// throw out extra transactions that were included by the institution responses
	filteredTxns := make([]Transaction, 0, len(allTxns))
	for _, t := range allTxns {
		if t.Date.Before(start) {
			continue
		}
		filteredTxns = append(filteredTxns, t)
	}
	allTxns = filteredTxns

	processTxns(allTxns)

	if err := ldg.AddTransactions(allTxns); err != nil {
		logger.Warn("Failed to add transactions to ledger", zap.Error(err))
		return err
	}
	logger.Info("Ledger successfully updated")
	return errs.ErrOrNil()
}

func min(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func (s *Store) startSync() (startedSync bool) {
	return s.syncing.CAS(false, true)
}

func (s *Store) stopSync(err error) {
	s.syncing.Store(false)
	s.lastSyncErr.Store(err)
	if err != nil {
		s.logger.Error("Error syncing", zap.Error(err))
	}
}

// SyncStatus returns whether sync is running and the most recent sync error
func (s *Store) SyncStatus() (syncing bool, lastSyncErr error) {
	return s.syncing.Load(), s.lastSyncErr.Load()
}

func currentDate() time.Time {
	return time.Now().UTC().Round(day)
}

// SyncRecent runs Sync for any new transactions since the last sync. Currently assumes last the last txn's date should be the start date.
func (s *Store) SyncRecent(download downloader, processTxns txnMutator) {
	now := currentDate()
	// TODO inline LastTransactionTime?
	// TODO use smart first date selection on a per-account basis
	lastTxnTime := s.Ledger.LastTransactionTime()
	if lastTxnTime.IsZero() {
		lastTxnTime = now.Add(-30 * day)
	}
	s.StartSync(lastTxnTime, now, download, processTxns)
}

// Resync runs Sync from the first date in the ledger until now
func (s *Store) Resync(download downloader, processTxns txnMutator) {
	now := currentDate()
	s.StartSync(s.Ledger.FirstTransactionTime(), now, download, processTxns)
}

// RenameAccount wraps ledger.RenameAccount and syncs changes to disk
func (s *Store) RenameAccount(oldName, newName, oldID, newID string) (int, error) {
	updatedCount := s.Ledger.RenameAccount(oldName, newName, oldID, newID)
	return updatedCount, s.syncFile()
}

// UpdateAccount wraps ledger.UpdateAccount and syncs changes to disk
func (s *Store) UpdateAccount(oldAccount, newAccount string) error {
	return pipe.OpFuncs{
		func() error { return s.Ledger.UpdateAccount(oldAccount, newAccount) },
		s.syncFile,
	}.Do()
}

// AddTransactions wraps ledger.AddTransactions and syncs changes to disk
func (s *Store) AddTransactions(txns []Transaction) error {
	return pipe.OpFuncs{
		func() error { return s.Ledger.AddTransactions(txns) },
		s.syncFile,
	}.Do()
}

// UpdateTransaction wraps ledger.UpdateTransaction and syncs changes to disk
func (s *Store) UpdateTransaction(id string, txn Transaction) error {
	return pipe.OpFuncs{
		func() error { return s.Ledger.UpdateTransaction(id, txn) },
		s.syncFile,
	}.Do()
}

// UpdateTransactions wraps ledger.UpdateTransactions and syncs changes to disk
func (s *Store) UpdateTransactions(txns map[string]Transaction) error {
	var ledgerErrs, errs sErrors.Errors
	for id, txn := range txns {
		switch err := s.Ledger.UpdateTransaction(id, txn).(type) {
		case Error:
			ledgerErrs.AddErr(err)
		default:
			errs.AddErr(err)
		}
	}
	return pipe.OpFuncs{
		errs.ErrOrNil,
		s.syncFile,          // sync file even if there are validation errors
		ledgerErrs.ErrOrNil, // return least critical errors last
	}.Do()
}

// UpdateOpeningBalance wraps ledger.UpdateOpeningBalance and syncs changes to disk
func (s *Store) UpdateOpeningBalance(opening Transaction) error {
	return pipe.OpFuncs{
		func() error { return s.Ledger.UpdateOpeningBalance(opening) },
		s.syncFile,
	}.Do()
}
