package db_experiments

import (
	"fmt"
	"os"
	"testing"
	"time"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/internal/test"
	"github.com/cometbft/cometbft/libs/rand"
	"github.com/docker/go-units"
)

var backends = []dbm.BackendType{
	dbm.CLevelDBBackend,
	dbm.RocksDBBackend,
	dbm.BadgerDBBackend,
	dbm.GoLevelDBBackend,
	dbm.BoltDBBackend,
}

//func BenchmarkSmallInserts(b *testing.B) {
//	keySize := 64
//	valueSize := 1 * units.MiB
//	for _, backend := range backends {
//		runBackendExperimentWithTimeOut(
//			inserts,
//			backend,
//			keySize,
//			valueSize,
//			15*time.Minute,
//			b,
//		)
//	}
//}
//
//func BenchmarkMediumInserts(b *testing.B) {
//	keySize := 64
//	valueSize := 64 * units.MiB
//	for _, backend := range backends {
//		runBackendExperimentWithTimeOut(
//			inserts,
//			backend,
//			keySize,
//			valueSize,
//			15*time.Minute,
//			b,
//		)
//	}
//}
//
//func BenchmarkLargeInserts(b *testing.B) {
//	keySize := 64
//	valueSize := 512 * units.MiB
//	for _, backend := range backends {
//		runBackendExperimentWithTimeOut(
//			inserts,
//			backend,
//			keySize,
//			valueSize,
//			15*time.Minute,
//			b,
//		)
//	}
//}
//
//func BenchmarkSmallDeletions(b *testing.B) {
//	keySize := 64
//	valueSize := 1 * units.MiB
//	for _, backend := range backends {
//		runBackendExperimentWithTimeOut(
//			deletions,
//			backend,
//			keySize,
//			valueSize,
//			15*time.Minute,
//			b,
//		)
//	}
//}
//
//func BenchmarkSmallBatchInserts(b *testing.B) {
//	keySize := 64
//	valueSize := 1 * units.MiB
//	for _, backend := range backends {
//		runBackendExperimentWithTimeOut(
//			batchInserts,
//			backend,
//			keySize,
//			valueSize,
//			15*time.Minute,
//			b,
//		)
//	}
//}
//
//func BenchmarkSmallBatchDeletions(b *testing.B) {
//	keySize := 64
//	valueSize := 1 * units.MiB
//	for _, backend := range backends {
//		runBackendExperimentWithTimeOut(
//			batchDeletions,
//			backend,
//			keySize,
//			valueSize,
//			15*time.Minute,
//			b,
//		)
//	}
//}
//
//func BenchmarkSmallFluctuations(b *testing.B) {
//	keySize := 64
//	valueSize := 1 * units.MiB
//	for _, backend := range backends {
//		runBackendExperimentWithTimeOut(
//			fluctuations,
//			backend,
//			keySize,
//			valueSize,
//			15*time.Minute,
//			b,
//		)
//	}
//}
//
//func BenchmarkFluctuationsSequentialKeys(b *testing.B) {
//	keySize := 64
//	valueSize := 1 * units.MiB
//	for _, backend := range backends {
//		runBackendExperimentWithTimeOut(
//			fluctuationsSequentialKeys,
//			backend,
//			keySize,
//			valueSize,
//			30*time.Minute,
//			b,
//		)
//	}
//}
//
//func BenchmarkFluctuationsSequentialKeysDeleteSync(b *testing.B) {
//	keySize := 64
//	valueSize := 1 * units.MiB
//	for _, backend := range backends {
//		runBackendExperimentWithTimeOut(
//			fluctuationsSequentialKeysDeleteSync,
//			backend,
//			keySize,
//			valueSize,
//			30*time.Minute,
//			b,
//		)
//	}
//}

func BenchmarkInsertsSequential(b *testing.B) {
	experiment := func(backendType dbm.BackendType) {
		config := test.ResetTestRoot("db_benchmark")
		defer func(path string) {
			err := os.RemoveAll(path)
			if err != nil {
				b.Fatal(err)
			}
		}(config.RootDir)

		db, err := dbm.NewDB("test", backendType, config.DBDir())
		if err != nil {
			b.Fatal(err)
		}

		totalSetDuration := 0 * time.Second
		nInserts := 5 * 1024
		for i := 0; i < nInserts; i++ {
			key := uint64ToBytes(uint64(i))
			value := rand.Bytes(1 * units.MiB)
			startPut := time.Now()
			err := db.Set(key, value)
			if err != nil {
				b.Fatal(err)
			}
			totalSetDuration += time.Since(startPut)
		}

		fmt.Println(fmt.Sprintf("%v\nTotal time to insert %v sequential keys: %v\nEventual dirSize: %v", backendType, nInserts, totalSetDuration, dirSize(config.DBDir())))
	}

	for _, backend := range backends {
		experiment(backend)
	}
}

func BenchmarkInsertsRandom(b *testing.B) {
	experiment := func(backendType dbm.BackendType) {
		config := test.ResetTestRoot("db_benchmark")
		defer func(path string) {
			err := os.RemoveAll(path)
			if err != nil {
				b.Fatal(err)
			}
		}(config.RootDir)

		db, err := dbm.NewDB("test", backendType, config.DBDir())
		if err != nil {
			b.Fatal(err)
		}

		totalSetDuration := 0 * time.Second
		nInserts := 5 * 1024
		for i := 0; i < nInserts; i++ {
			key := rand.Bytes(8)
			value := rand.Bytes(1 * units.MiB)
			startPut := time.Now()
			err := db.Set(key, value)
			if err != nil {
				b.Fatal(err)
			}
			totalSetDuration += time.Since(startPut)
		}

		fmt.Println(fmt.Sprintf("%v\nTotal time to insert %v random keys: %v\nEventual dirSize: %v", backendType, nInserts, totalSetDuration, dirSize(config.DBDir())))
	}

	for _, backend := range backends {
		experiment(backend)
	}
}
