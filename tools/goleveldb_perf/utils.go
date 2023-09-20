package db_experiments

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/internal/test"
	"github.com/cometbft/cometbft/libs/rand"
	"github.com/docker/go-units"
)

// Step is used to keep track of the steps taken during the test.
// Describes the size of the db and the number of records in it at various
// points during a test
type Step struct {
	// The name of the step
	Name string
	// The size of the db in kb
	Size int64
	// The number of records in the db
	Records int
	// The time step took
	Duration time.Duration
	// Sys memory used by the process
	SysMem uint64
}

type StepOptions struct {
	LastInserted uint64
	LastDeleted  uint64
}

// Performs one step as part of a test.
//
// The step is defined by `stepType`: `delete` or `insert`.
// The `count` defines the number of records to delete or insert.
// The `db` is the db to perform the step on.
//
// Returns the step that was performed.
func step(
	stepType string,
	count int,
	db dbm.DB,
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
		iter, err := db.Iterator(nil, nil)
		if err != nil {
			panic(fmt.Errorf("error calling Iterator(): %w", err))
		}
		defer iter.Close()

		deleted := 0
	iterating:
		for ; iter.Valid(); iter.Next() {
			select {
			case <-ctx.Done(): // we control if a step should terminate due to timeout
				return Step{Name: "timeout"}
			default:
				if err := db.Delete(iter.Key()); err != nil {
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
				if err := db.Delete(uint64ToBytes(curKey)); err != nil {
					panic(fmt.Errorf("error during Delete(): %w", err))
				}
			}
		}
		err := db.DeleteSync(uint64ToBytes(options.LastDeleted + uint64(count)))
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
				if err := db.Set(rand.Bytes(keySize), rand.Bytes(valueSize)); err != nil {
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
				if err := db.Set(uint64ToBytes(lastKeyInserted+1), rand.Bytes(valueSize)); err != nil {
					panic(fmt.Errorf("error during Set(): %w", err))
				}
				lastKeyInserted += 1
			}
		}
	} else if stepType == "batchInsert" {
		batch := db.NewBatch()
		defer func(batch dbm.Batch) {
			err := batch.Close()
			if err != nil {
				panic(err)
			}
		}(batch)
		for i := 0; i < count; i++ {
			select {
			case <-ctx.Done(): // we control if a step should terminate due to timeout
				return Step{Name: "timeout"}
			default:
				if err := batch.Set(rand.Bytes(keySize), rand.Bytes(valueSize)); err != nil {
					panic(fmt.Errorf("error during batch Set(): %w", err))
				}
			}
		}
		err := batch.WriteSync()
		if err != nil {
			panic(fmt.Errorf("error during bathc WriteSync(): %w", err))
		}
	} else if stepType == "batchDelete" {
		// Handle the deletion of records
		// Iterate over the db and delete the first `count` records
		iter, err := db.Iterator(nil, nil)
		if err != nil {
			panic(fmt.Errorf("error calling Iterator(): %w", err))
		}
		defer func(iter dbm.Iterator) {
			err := iter.Close()
			if err != nil {
				panic(err)
			}
		}(iter)

		batch := db.NewBatch()
		defer func(batch dbm.Batch) {
			err := batch.Close()
			if err != nil {
				panic(err)
			}
		}(batch)

		deleted := 0
	iteratingBatchDelete:
		for ; iter.Valid(); iter.Next() {
			select {
			case <-ctx.Done(): // we control if a step should terminate due to timeout
				return Step{Name: "timeout"}
			default:
				if err := batch.Delete(iter.Key()); err != nil {
					panic(fmt.Errorf("error during Delete(): %w", err))
				}
				deleted++
				if deleted == count {
					break iteratingBatchDelete
				}
			}
		}
		err = batch.WriteSync()
		if err != nil {
			panic(fmt.Errorf("error during batch WriteSync(): %w", err))
		}
	} else {
		panic("invalid step type")
	}

	return Step{
		Name: stepType,
		Size: dirSize(dbPath),
		// We don't want to call dbCount since for many backends it causes intense memory overuse
		Records:  dbCount(db),
		Duration: time.Since(curTime),
		SysMem:   getSysMem(),
	}
}

func getSysMem() uint64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return memStats.Sys / units.MiB
}

// Handles the removal of all db files.
// Should be called at the beginning of any test.
func dbPathRemove(dbPath string) {
	err := os.RemoveAll(dbPath)
	if err != nil {
		panic(fmt.Errorf("error removing the db directory: %w", err))
	}
}

// Counts the number of records in the `db`.
func dbCount(db dbm.DB) int {
	iter, err := db.Iterator(nil, nil)
	if err != nil {
		panic(fmt.Errorf("error calling Iterator(): %w", err))
	}
	defer iter.Close()
	iterCount := 0
	for ; iter.Valid(); iter.Next() {
		iterCount++
	}
	return iterCount
}

// Computes the size of the given `path` directory in Megabytes.
// Panics if any error occurs.
func dirSize(path string) int64 {
	err := fmt.Errorf("empty error")
	deadline := time.Now().Add(time.Second * 5)
	var size int64
	for err != nil && time.Now().Before(deadline) {
		size = 0
		err = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				size += info.Size()
			}
			return err
		})
	}
	if err != nil {
		panic(err)
	}
	return size / units.MiB
}

// PrintSteps prints the content of an array of `Step` structs.
func PrintSteps(steps []Step, experimentName string, backendType dbm.BackendType) {
	path := filepath.Join("measurements", fmt.Sprintf("%v_%v.txt", experimentName, backendType))
	file, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)

	_, err = file.WriteString(fmt.Sprintf("Backend: %v\nSteps:\n\t%15s%15s%15s%15s%15s\n", backendType, "name", "size (mb)", "records #", "duration", "sysmem (mb)"))
	if err != nil {
		panic(err)
	}
	for _, step := range steps {
		_, err := file.WriteString(
			fmt.Sprintf("\t%15v%15v%15v%15.2f%15v\n", step.Name, step.Size, step.Records, step.Duration.Seconds(), step.SysMem))
		if err != nil {
			panic(err)
		}
	}
}

func runWithTimeOut(timeout time.Duration, call func(ctx context.Context)) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	flagChan := make(chan int)

	go func() {
		call(ctx)
		flagChan <- 0
	}()

	select {
	case <-flagChan:
		return true
	case <-ctx.Done():
		return false
	}
}

func runBackendExperimentWithTimeOut(
	experiment func(backendType dbm.BackendType, keySize int, valueSize int, dbPath string, ctx context.Context) []Step,
	backend dbm.BackendType,
	keySize int,
	valueSize int,
	timeout time.Duration,
	b *testing.B) {
	config := test.ResetTestRoot("db_benchmark")
	var wg sync.WaitGroup
	wg.Add(1)
	success := runWithTimeOut(timeout, func(ctx context.Context) {
		defer wg.Done()
		steps := experiment(backend, keySize, valueSize, config.DBDir(), ctx)
		PrintSteps(steps, b.Name(), backend)
	})
	wg.Wait()
	if err := os.RemoveAll(config.RootDir); err != nil {
		panic(err)
	}
	if success {
		b.Log(fmt.Sprintf("Done with %v", backend))
	} else {
		b.Log(fmt.Sprintf("%v timed out", backend))
	}
}

func fillStorageToVolume(targetVolume, keySize, valueSize int, db dbm.DB) {
	recordingsPerStep := 1 * units.GiB / (keySize + valueSize)
	volumePerStep := recordingsPerStep * (keySize + valueSize)
	nFullSteps := targetVolume / volumePerStep
	remainingVolume := targetVolume % volumePerStep
	nRemainingRecordings := remainingVolume / (keySize + valueSize)

	oneStep := func(nRecordings int) {
		batch := db.NewBatch()
		defer func(batch dbm.Batch) {
			err := batch.Close()
			if err != nil {
				panic(err)
			}
		}(batch)
		for i := 0; i < nRecordings; i++ {
			if err := batch.Set(rand.Bytes(keySize), rand.Bytes(valueSize)); err != nil {
				panic(fmt.Errorf("error during batch Set(): %w", err))
			}
		}
		err := batch.WriteSync()
		if err != nil {
			panic(fmt.Errorf("error during bathc WriteSync(): %w", err))
		}
	}

	for i := 0; i < nFullSteps; i++ {
		oneStep(recordingsPerStep)
	}

	oneStep(nRemainingRecordings)
}

func uint64ToBytes(v uint64) []byte {
	byteSlice := make([]byte, 8) // 8 bytes for a uint64
	binary.BigEndian.PutUint64(byteSlice, v)
	return byteSlice
}

func bytesToUint64(v []byte) uint64 {
	if len(v) != 8 {
		panic(fmt.Errorf("should be 8 bytes"))
	}
	return binary.BigEndian.Uint64(v)
}

func fillStorageToVolumeSequentialKeys(targetVolume, valueSize int, db dbm.DB) {
	keySize := 64
	recordingsPerStep := 1 * units.GiB / (keySize + valueSize)
	volumePerStep := recordingsPerStep * (keySize + valueSize)
	nFullSteps := targetVolume / volumePerStep
	remainingVolume := targetVolume % volumePerStep
	nRemainingRecordings := remainingVolume / (keySize + valueSize)

	lastKey := uint64(0)
	oneStep := func(nRecordings int) {
		batch := db.NewBatch()
		defer func(batch dbm.Batch) {
			err := batch.Close()
			if err != nil {
				panic(err)
			}
		}(batch)
		for i := 0; i < nRecordings; i++ {
			if err := batch.Set(uint64ToBytes(lastKey), rand.Bytes(valueSize)); err != nil {
				panic(fmt.Errorf("error during batch Set(): %w", err))
			}
			lastKey++
		}
		err := batch.WriteSync()
		if err != nil {
			panic(fmt.Errorf("error during bathc WriteSync(): %w", err))
		}
	}

	for i := 0; i < nFullSteps; i++ {
		oneStep(recordingsPerStep)
	}

	oneStep(nRemainingRecordings)
}
