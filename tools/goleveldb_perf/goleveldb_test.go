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
	"github.com/cometbft/cometbft/libs/rand"
	"github.com/docker/go-units"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func TestGoLevelDBCompactionSequential(t *testing.T) {
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
			insertStep := step("insert", recordsPerCycle, db, 64, 1*units.MiB, config.DBDir(), context.Background(), StepOptions{})
			steps = append(steps, insertStep)

			deleteStep := step("delete", recordsPerCycle, db, 64, 1*units.MiB, config.DBDir(), context.Background(), StepOptions{})
			steps = append(steps, deleteStep)
			fmt.Println(deleteStep)
		}
		return steps
	}

	nCycles := 100
	recordsPerCycleList := []int{256}
	for _, recordsPerCycle := range recordsPerCycleList {
		steps := experiment(nCycles, recordsPerCycle)
		PrintSteps(steps, fmt.Sprintf("%s_%v", t.Name(), recordsPerCycle), dbm.GoLevelDBBackend)
	}
}

func TestGoLevelDBCompactionSequentialRangeCompact(t *testing.T) {
	experiment := func(nCycles, recordsPerCycle int) []Step {
		config := test.ResetTestRoot("db_benchmark")
		defer func(path string) {
			err := os.RemoveAll(path)
			require.NoError(t, err)
		}(config.RootDir)

		db, err := leveldb.OpenFile(config.DBDir(), nil)

		require.NoError(t, err)
		var steps []Step
		for i := 0; i < nCycles; i++ {
			if i%20 == 0 && i > 0 {
				fmt.Println(i, recordsPerCycle)
			}
			lastKeyInserted := uint64(i*recordsPerCycle) + math.MaxUint64

			steps = append(steps, stepGoLevelDB("insertSequential", recordsPerCycle, db, 64, 1*units.MiB, config.DBDir(), context.Background(), StepOptions{LastInserted: lastKeyInserted}))

			steps = append(steps, stepGoLevelDB("deleteSequentialCompactRange", recordsPerCycle, db, 64, 1*units.MiB, config.DBDir(), context.Background(), StepOptions{LastDeleted: lastKeyInserted}))
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

func stepGoLevelDB(
	stepType string,
	count int,
	db *leveldb.DB,
	keySize int,
	valueSize int,
	dbPath string,
	ctx context.Context,
	options StepOptions,
) Step {
	curTime := time.Now()
	if stepType == "delete" {
		// Handle the deletion of records
		// Iterate over the db and delete the first `count` records
		iter := db.NewIterator(&util.Range{}, nil)
		iter.Seek([]byte{})
		defer iter.Release()

		deleted := 0
	iterating:
		for ; iter.Valid(); iter.Next() {
			select {
			case <-ctx.Done(): // we control if a step should terminate due to timeout
				return Step{Name: "timeout"}
			default:
				if err := db.Delete(iter.Key(), nil); err != nil {
					panic(fmt.Errorf("error during Delete(): %w", err))
				}
				deleted++
				if deleted == count {
					break iterating
				}
			}
		}
	} else if stepType == "deleteSequential" {
		for curKey := options.LastDeleted + 1; curKey < options.LastDeleted+uint64(count); curKey++ {
			select {
			case <-ctx.Done(): // we control if a step should terminate due to timeout
				return Step{Name: "timeout"}
			default:
				if err := db.Delete(uint64ToBytes(curKey), nil); err != nil {
					panic(fmt.Errorf("error during Delete(): %w", err))
				}
			}
		}
		err := db.Delete(uint64ToBytes(options.LastDeleted+uint64(count)), &opt.WriteOptions{Sync: true})
		if err != nil {
			panic(err)
		}
	} else if stepType == "deleteSequentialCompactRange" {
		for curKey := options.LastDeleted + 1; curKey < options.LastDeleted+uint64(count); curKey++ {
			select {
			case <-ctx.Done(): // we control if a step should terminate due to timeout
				return Step{Name: "timeout"}
			default:
				if err := db.Delete(uint64ToBytes(curKey), nil); err != nil {
					panic(fmt.Errorf("error during Delete(): %w", err))
				}
			}
		}
		err := db.Delete(uint64ToBytes(options.LastDeleted+uint64(count)), &opt.WriteOptions{Sync: true})
		if err != nil {
			panic(err)
		}
		err = db.CompactRange(util.Range{Start: uint64ToBytes(options.LastDeleted), Limit: uint64ToBytes(options.LastDeleted + uint64(count+1))})
		if err != nil {
			panic(err)
		}
	} else if stepType == "insert" {
		// Handle the insertion of records
		// Write `count` records to the db
		for i := 0; i < count; i++ {
			select {
			case <-ctx.Done(): // we control if a step should terminate due to timeout
				return Step{Name: "timeout"}
			default:
				if err := db.Put(rand.Bytes(keySize), rand.Bytes(valueSize), nil); err != nil {
					panic(fmt.Errorf("error during Set(): %w", err))
				}
			}
		}
	} else if stepType == "insertSequential" {
		// Handle the insertion of records
		// Write `count` records to the db
		lastKeyInserted := options.LastInserted
		for i := 0; i < count; i++ {
			select {
			case <-ctx.Done(): // we control if a step should terminate due to timeout
				return Step{Name: "timeout"}
			default:
				if err := db.Put(uint64ToBytes(lastKeyInserted+1), rand.Bytes(valueSize), nil); err != nil {
					panic(fmt.Errorf("error during Set(): %w", err))
				}
				lastKeyInserted += 1
			}
		}
	} else {
		panic("invalid step type")
	}

	return Step{
		Name:     stepType,
		Size:     dirSize(dbPath),
		Records:  dbCountGoLevelDB(db),
		Duration: time.Since(curTime),
		SysMem:   getSysMem(),
	}
}

func dbCountGoLevelDB(db *leveldb.DB) int {
	iter := db.NewIterator(&util.Range{}, nil)
	iter.Seek([]byte{})
	defer iter.Release()
	iterCount := 0
	for ; iter.Valid(); iter.Next() {
		iterCount++
	}
	return iterCount
}
