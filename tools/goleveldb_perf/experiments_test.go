package db_experiments

import (
	"os"
	"testing"
	"time"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/internal/test"
	"github.com/cometbft/cometbft/libs/rand"
	"github.com/docker/go-units"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
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
	for _, backend := range backends {
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

func BenchmarkFluctuationsSequentialKeysDeleteSync(b *testing.B) {
	keySize := 64
	valueSize := 1 * units.MiB
	for _, backend := range backends {
		runBackendExperimentWithTimeOut(
			fluctuationsSequentialKeysDeleteSync,
			backend,
			keySize,
			valueSize,
			30*time.Minute,
			b,
		)
	}
}

func TestGoLevelDBCompaction(t *testing.T) {
	config := test.ResetTestRoot("db_benchmark")
	defer func(path string) {
		err := os.RemoveAll(path)
		require.NoError(t, err)
	}(config.RootDir)

	db, err := leveldb.OpenFile(config.DBDir(), &opt.Options{})
	require.NoError(t, err)
	defer func(db *leveldb.DB) {
		err := db.Close()
		require.NoError(t, err)
	}(db)

	nCycles := 10
	recordsPerCycle := uint64(16)
	for i := 0; i < nCycles; i++ {
		firstKey := uint64(i) * recordsPerCycle
		for key := firstKey; key < firstKey+recordsPerCycle; key++ {
			err := db.Put(uint64ToBytes(key), rand.Bytes(1*units.MiB), nil)
			require.NoError(t, err)
		}

		for key := firstKey; key < firstKey+recordsPerCycle; key++ {
			err := db.Delete(uint64ToBytes(key), nil)
			require.NoError(t, err)
		}

		require.True(t, dirSize(config.DBDir()) < 20)
	}
}
