package db_experiments

import (
	"fmt"
	"os"
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/internal/test"
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
		config := test.ResetTestRoot("db_benchmark")
		steps := inserts(backend, keySize, valueSize, config.DBDir())
		PrintSteps(steps, b.Name(), backend)
		if err := os.RemoveAll(config.RootDir); err != nil {
			panic(err)
		}
		b.Log(fmt.Sprintf("Done with %v", backend))
	}
}

func BenchmarkMediumInserts(b *testing.B) {
	keySize := 64
	valueSize := 64 * units.MiB
	for _, backend := range backends {
		config := test.ResetTestRoot("db_benchmark")
		steps := inserts(backend, keySize, valueSize, config.DBDir())
		PrintSteps(steps, b.Name(), backend)
		if err := os.RemoveAll(config.RootDir); err != nil {
			panic(err)
		}
		b.Log(fmt.Sprintf("Done with %v", backend))
	}
}

func BenchmarkLargeInserts(b *testing.B) {
	keySize := 64
	valueSize := 512 * units.MiB
	for _, backend := range backends {
		config := test.ResetTestRoot("db_benchmark")
		steps := inserts(backend, keySize, valueSize, config.DBDir())
		PrintSteps(steps, b.Name(), backend)
		if err := os.RemoveAll(config.RootDir); err != nil {
			panic(err)
		}
		b.Log(fmt.Sprintf("Done with %v", backend))
	}
}

func BenchmarkSmallDeletions(b *testing.B) {
	keySize := 64
	valueSize := 1 * units.MiB
	for _, backend := range backends {
		config := test.ResetTestRoot("db_benchmark")
		steps := deletions(backend, keySize, valueSize, config.DBDir())
		PrintSteps(steps, b.Name(), backend)
		if err := os.RemoveAll(config.RootDir); err != nil {
			panic(err)
		}
		b.Log(fmt.Sprintf("Done with %v", backend))
	}
}
