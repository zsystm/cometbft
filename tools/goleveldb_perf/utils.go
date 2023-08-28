package db_experiments

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	dbm "github.com/cometbft/cometbft-db"
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
		for ; iter.Valid(); iter.Next() {
			if err := db.Delete(iter.Key()); err != nil {
				panic(fmt.Errorf("error during Delete(): %w", err))
			}
			deleted++
			if deleted == count {
				fmt.Printf("\n\t\t\t---- deleted %v records", count)
				break
			}
		}
	} else if stepType == "insert" {
		// Handle the insertion of records
		// Write `count` records to the db
		for i := 0; i < count; i++ {
			if err := db.Set(rand.Bytes(keySize), rand.Bytes(valueSize)); err != nil {
				panic(fmt.Errorf("error during Set(): %w", err))
			}
		}
	} else {
		panic("invalid step type")
	}

	return Step{
		Name:     stepType,
		Size:     dirSize(dbPath),
		Records:  dbCount(db),
		Duration: time.Since(curTime),
	}
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

// Computes the size of the given `path` directory in bytes.
// Panics if any error occurs.
func dirSize(path string) int64 {
	err := fmt.Errorf("empty error")
	deadline := time.Now().Add(time.Second * 5)
	var size int64
	for err != nil && time.Now().Before(deadline) {
		size = 0
		err = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Println(err)
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

	_, err = file.WriteString(fmt.Sprintf("Backend: %v\nSteps:\n\t%15s%15s%15s\n", backendType, "name", "size (mb)", "records #"))
	if err != nil {
		panic(err)
	}
	for _, step := range steps {
		_, err := file.WriteString(fmt.Sprintf("\t%15v\n", step))
		if err != nil {
			panic(err)
		}
	}
}
