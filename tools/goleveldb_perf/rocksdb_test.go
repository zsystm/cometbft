package db_experiments

import (
	"context"
	"fmt"
	"math"
	"os"
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/internal/test"
	"github.com/docker/go-units"
	"github.com/stretchr/testify/require"
	"github.com/tecbot/gorocksdb"
)

func TestRocksDBCompaction(t *testing.T) {
	experiment := func(nCycles, recordsPerCycle int) []Step {
		config := test.ResetTestRoot("db_benchmark")
		defer func(path string) {
			err := os.RemoveAll(path)
			require.NoError(t, err)
		}(config.RootDir)

		opts := gorocksdb.NewDefaultOptions()
		opts.SetCreateIfMissing(true)
		db, err := gorocksdb.OpenDb(opts, config.DBDir())
		require.NoError(t, err)
		var steps []Step
		for i := 0; i < nCycles; i++ {
			insertStep := stepRocksDB("insert", recordsPerCycle, db, 64, 1*units.MiB, config.DBDir(), context.Background(), StepOptions{})
			steps = append(steps, insertStep)

			deleteStep := stepRocksDB("delete", recordsPerCycle, db, 64, 1*units.MiB, config.DBDir(), context.Background(), StepOptions{})
			steps = append(steps, deleteStep)
			fmt.Println(deleteStep)
		}
		return steps
	}

	nCycles := 100
	recordsPerCycleList := []int{256}
	for _, recordsPerCycle := range recordsPerCycleList {
		steps := experiment(nCycles, recordsPerCycle)
		PrintSteps(steps, fmt.Sprintf("%s_%v", t.Name(), recordsPerCycle), dbm.RocksDBBackend)
	}
}

func TestRocksDBCompactionSequentialRangeCompact(t *testing.T) {
	experiment := func(nCycles, recordsPerCycle int) []Step {
		config := test.ResetTestRoot("db_benchmark")
		defer func(path string) {
			err := os.RemoveAll(path)
			require.NoError(t, err)
		}(config.RootDir)

		opts := gorocksdb.NewDefaultOptions()
		opts.SetCreateIfMissing(true)
		db, err := gorocksdb.OpenDb(opts, config.DBDir())
		require.NoError(t, err)
		var steps []Step
		for i := 0; i < nCycles; i++ {
			if i%20 == 0 && i > 0 {
				fmt.Println(i, recordsPerCycle)
			}
			lastKeyInserted := uint64(i*recordsPerCycle) + math.MaxUint64

			steps = append(steps, stepRocksDB("insertSequential", recordsPerCycle, db, 64, 1*units.MiB, config.DBDir(), context.Background(), StepOptions{LastInserted: lastKeyInserted}))

			steps = append(steps, stepRocksDB("deleteSequentialCompactRange", recordsPerCycle, db, 64, 1*units.MiB, config.DBDir(), context.Background(), StepOptions{LastDeleted: lastKeyInserted}))
		}
		return steps
	}

	nCycles := 100
	recordsPerCycleList := []int{256}
	for _, recordsPerCycle := range recordsPerCycleList {
		steps := experiment(nCycles, recordsPerCycle)
		PrintSteps(steps, fmt.Sprintf("%s_%v", t.Name(), recordsPerCycle), dbm.RocksDBBackend)
	}
}

func TestName(t *testing.T) {
	config := test.ResetTestRoot("db_benchmark")
	defer func(path string) {
		err := os.RemoveAll(path)
		require.NoError(t, err)
	}(config.RootDir)

	opts := gorocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	db, err := gorocksdb.OpenDb(opts, config.DBDir())
	require.NoError(t, err)
	err = db.Put(gorocksdb.NewDefaultWriteOptions(), []byte{228}, []byte{144})
	err = db.Put(gorocksdb.NewDefaultWriteOptions(), []byte{32}, []byte{144})
	require.NoError(t, err)
	fmt.Println(dbCountRocksDB(db))
	fmt.Println(stepRocksDB("delete", 1, db, 64, 64, config.DBDir(), context.Background(), StepOptions{}))
}
