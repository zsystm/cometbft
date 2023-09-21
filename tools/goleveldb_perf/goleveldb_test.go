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

func TestLongFluctuations(t *testing.T) {
	config := test.ResetTestRoot("db_benchmark")
	defer func(path string) {
		err := os.RemoveAll(path)
		require.NoError(t, err)
	}(config.RootDir)

	db, err := leveldb.OpenFile(config.DBDir(), nil)
	require.NoError(t, err)

	lastKeyInserted := fillGoLevelDBStorageToVolumeSequentialKeys(5*units.GiB, 1*units.MiB, db)
	fmt.Println("Filled to initial volume")

	startTime := time.Now()
	var steps []Step
	recordsPerCycle := 1024
	for i := 0; time.Since(startTime) < time.Hour; i++ {
		steps = append(steps, stepGoLevelDB("insertSequential", recordsPerCycle, db, 64, 1*units.MiB, config.DBDir(), context.Background(), StepOptions{LastInserted: lastKeyInserted}))
		steps = append(steps, stepGoLevelDB("deleteSequentialCompactRange", recordsPerCycle, db, 64, 1*units.MiB, config.DBDir(), context.Background(), StepOptions{LastDeleted: lastKeyInserted}))
		lastKeyInserted += uint64(recordsPerCycle)
		if i%10 == 0 && i > 0 {
			fmt.Println(fmt.Sprintf("Done %v steps; Last step: %v", i, steps[len(steps)-1]))
			PrintSteps(steps, t.Name(), dbm.GoLevelDBBackend)
		}
	}
}

func BenchmarkCompactionOverheadSequential(b *testing.B) {
	config := test.ResetTestRoot("db_benchmark")
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			b.Fatal(err)
		}
	}(config.RootDir)

	db, err := leveldb.OpenFile(config.DBDir(), nil)
	if err != nil {
		b.Fatal(err)
	}

	totalPutDuration := 0 * time.Second
	totalCompactionDuration := 0 * time.Second
	nRounds := 10
	nInsertsPerRound := 1024
	for i := 0; i < nRounds; i++ {
		// insert range
		for j := 0; j < nInsertsPerRound; j++ {
			key := uint64ToBytes(uint64(i*nInsertsPerRound + j))
			value := rand.Bytes(1 * units.MiB)
			startPut := time.Now()
			err := db.Put(key, value, nil)
			if err != nil {
				b.Fatal(err)
			}
			totalPutDuration += time.Since(startPut)
		}
		// delete range
		for j := 0; j < nInsertsPerRound; j++ {
			key := uint64ToBytes(uint64(i*nInsertsPerRound + j))
			startPut := time.Now()
			err := db.Delete(key, nil)
			if err != nil {
				b.Fatal(err)
			}
			totalPutDuration += time.Since(startPut)
		}
		//compact range
		rangeToCompact := util.Range{
			Start: uint64ToBytes(uint64(i * nInsertsPerRound)),
			Limit: uint64ToBytes(uint64((i + 1) * nInsertsPerRound)),
		}
		startCompact := time.Now()
		err := db.CompactRange(rangeToCompact)
		totalCompactionDuration += time.Since(startCompact)
		if err != nil {
			b.Fatal(err)
		}
	}
	fmt.Println(fmt.Sprintf("Compaction duration: %v\nPut duration: %v\nDirSize: %v", totalCompactionDuration, totalPutDuration, dirSize(config.DBDir())))
}
