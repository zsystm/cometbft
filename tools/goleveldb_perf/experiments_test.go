package db_experiments

import (
	"context"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/internal/test"
	"github.com/docker/go-units"
	"github.com/stretchr/testify/require"
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
	experiment := func(nCycles, recordsPerCycle int) []Step {
		config := test.ResetTestRoot("db_benchmark")
		defer func(path string) {
			err := os.RemoveAll(path)
			require.NoError(t, err)
		}(config.RootDir)

		db, err := dbm.NewDB("test_db", dbm.GoLevelDBBackend, config.DBDir())
		require.NoError(t, err)
		var steps []Step
		for i := 0; i < nCycles; i++ {
			if i%20 == 0 && i > 0 {
				fmt.Println(i, recordsPerCycle)
			}
			lastKeyInserted := uint64(i*recordsPerCycle) + math.MaxUint64

			steps = append(steps, step("insertSequential", recordsPerCycle, db, 64, 1*units.MiB, config.DBDir(), context.Background(), StepOptions{LastInserted: lastKeyInserted}))

			steps = append(steps, step("deleteSequential", recordsPerCycle, db, 64, 1*units.MiB, config.DBDir(), context.Background(), StepOptions{LastDeleted: lastKeyInserted}))
		}
		return steps
	}

	nCycles := 100
	recordsPerCycleList := []int{64, 256, 1024}
	for _, recordsPerCycle := range recordsPerCycleList {
		steps := experiment(nCycles, recordsPerCycle)
		PrintSteps(steps, fmt.Sprintf("%s_%v", t.Name(), recordsPerCycle), dbm.GoLevelDBBackend)
	}
}
