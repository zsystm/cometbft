package db_experiments

import (
	"testing"
	"time"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/docker/go-units"
)

var backends = []dbm.BackendType{
	dbm.CLevelDBBackend,
	dbm.RocksDBBackend,
	dbm.BadgerDBBackend,
	dbm.GoLevelDBBackend,
	dbm.BoltDBBackend,
}

func BenchmarkSmallInserts(b *testing.B) {
	keySize := 64
	valueSize := 1 * units.MiB
	for _, backend := range backends {
		runBackendExperimentWithTimeOut(
			inserts,
			backend,
			keySize,
			valueSize,
			15*time.Minute,
			b,
		)
	}
}

func BenchmarkMediumInserts(b *testing.B) {
	keySize := 64
	valueSize := 64 * units.MiB
	for _, backend := range backends {
		runBackendExperimentWithTimeOut(
			inserts,
			backend,
			keySize,
			valueSize,
			15*time.Minute,
			b,
		)
	}
}

func BenchmarkLargeInserts(b *testing.B) {
	keySize := 64
	valueSize := 512 * units.MiB
	for _, backend := range backends {
		runBackendExperimentWithTimeOut(
			inserts,
			backend,
			keySize,
			valueSize,
			15*time.Minute,
			b,
		)
	}
}

func BenchmarkSmallDeletions(b *testing.B) {
	keySize := 64
	valueSize := 1 * units.MiB
	for _, backend := range backends {
		runBackendExperimentWithTimeOut(
			deletions,
			backend,
			keySize,
			valueSize,
			15*time.Minute,
			b,
		)
	}
}

func BenchmarkSmallBatchInserts(b *testing.B) {
	keySize := 64
	valueSize := 1 * units.MiB
	for _, backend := range backends {
		runBackendExperimentWithTimeOut(
			batchInserts,
			backend,
			keySize,
			valueSize,
			15*time.Minute,
			b,
		)
	}
}

func BenchmarkSmallBatchDeletions(b *testing.B) {
	keySize := 64
	valueSize := 1 * units.MiB
	for _, backend := range backends {
		runBackendExperimentWithTimeOut(
			batchDeletions,
			backend,
			keySize,
			valueSize,
			15*time.Minute,
			b,
		)
	}
}

func BenchmarkSmallFluctuations(b *testing.B) {
	keySize := 64
	valueSize := 1 * units.MiB
	for _, backend := range backends {
		runBackendExperimentWithTimeOut(
			fluctuations,
			backend,
			keySize,
			valueSize,
			15*time.Minute,
			b,
		)
	}
}

func BenchmarkFluctuationsSequentialKeys(b *testing.B) {
	keySize := 64
	valueSize := 1 * units.MiB
	for _, backend := range []dbm.BackendType{
		dbm.BadgerDBBackend,
		dbm.GoLevelDBBackend,
		dbm.BoltDBBackend,
	} {
		runBackendExperimentWithTimeOut(
			fluctuationsSequentialKeys,
			backend,
			keySize,
			valueSize,
			30*time.Minute,
			b,
		)
	}
}
