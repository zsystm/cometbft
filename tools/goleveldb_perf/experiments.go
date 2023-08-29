package db_experiments

import (
	"context"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/docker/go-units"
)

func inserts(backendType dbm.BackendType, keySize int, valueSize int, dbPath string, ctx context.Context) []Step {
	targetStorageSize := 10 * units.GiB

	db, err := dbm.NewDB("experiment_db", backendType, dbPath)
	if err != nil {
		panic(err)
	}
	defer func(db dbm.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(db)

	var steps []Step
	steps = append(steps, Step{
		Name:     "initial",
		Size:     dirSize(dbPath),
		Records:  dbCount(db),
		Duration: 0,
		SysMem:   getSysMem(),
	})

	currentStorageSize := 0
	recordingsPerStep := 1 * units.GiB / (keySize + valueSize)
	for currentStorageSize < targetStorageSize {
		select {
		case <-ctx.Done():
			return steps
		default:
			steps = append(steps, step("insert", recordingsPerStep, db, keySize, valueSize, dbPath, ctx))
			currentStorageSize += (keySize + valueSize) * recordingsPerStep

			// make sure process is not killed due to the overuse of memory
			if getSysMem()*units.MiB > 5*units.GiB {
				steps = append(steps, Step{Name: "memOveruse"})
				return steps
			}
		}
	}

	return steps
}

func deletions(backendType dbm.BackendType, keySize int, valueSize int, dbPath string, ctx context.Context) []Step {
	initialStorageSize := 10 * units.GiB

	db, err := dbm.NewDB("experiment_db", backendType, dbPath)
	if err != nil {
		panic(err)
	}
	defer func(db dbm.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(db)

	var steps []Step
	initialRecordings := initialStorageSize / (keySize + valueSize)
	steps = append(steps, step("insert", initialRecordings, db, keySize, valueSize, dbPath, ctx))

	recordingsPerStep := 1 * units.GiB / (keySize + valueSize)
	currentStorageSize := (keySize + valueSize) * initialRecordings
	for currentStorageSize > 0 {
		select {
		case <-ctx.Done():
			return steps
		default:
			steps = append(steps, step("delete", recordingsPerStep, db, keySize, valueSize, dbPath, ctx))
			currentStorageSize -= (keySize + valueSize) * recordingsPerStep
		}
	}

	return steps
}

func batchInserts(backendType dbm.BackendType, keySize int, valueSize int, dbPath string, ctx context.Context) []Step {
	targetStorageSize := 10 * units.GiB

	db, err := dbm.NewDB("experiment_db", backendType, dbPath)
	if err != nil {
		panic(err)
	}
	defer func(db dbm.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(db)

	var steps []Step
	steps = append(steps, Step{
		Name:     "initial",
		Size:     dirSize(dbPath),
		Records:  dbCount(db),
		Duration: 0,
		SysMem:   getSysMem(),
	})

	currentStorageSize := 0
	recordingsPerStep := 1 * units.GiB / (keySize + valueSize)
	for currentStorageSize < targetStorageSize {
		select {
		case <-ctx.Done():
			return steps
		default:
			steps = append(steps, step("batchInsert", recordingsPerStep, db, keySize, valueSize, dbPath, ctx))
			currentStorageSize += (keySize + valueSize) * recordingsPerStep
		}
	}

	return steps
}

func batchDeletions(backendType dbm.BackendType, keySize int, valueSize int, dbPath string, ctx context.Context) []Step {
	initialStorageSize := 10 * units.GiB

	db, err := dbm.NewDB("experiment_db", backendType, dbPath)
	if err != nil {
		panic(err)
	}
	defer func(db dbm.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(db)

	var steps []Step
	initialRecordings := initialStorageSize / (keySize + valueSize)
	steps = append(steps, step("insert", initialRecordings, db, keySize, valueSize, dbPath, ctx))

	recordingsPerStep := 1 * units.GiB / (keySize + valueSize)
	currentStorageSize := (keySize + valueSize) * initialRecordings
	for currentStorageSize > 0 {
		select {
		case <-ctx.Done():
			return steps
		default:
			steps = append(steps, step("batchDelete", recordingsPerStep, db, keySize, valueSize, dbPath, ctx))
			currentStorageSize -= (keySize + valueSize) * recordingsPerStep
		}
	}

	return steps
}

func fluctuations(backendType dbm.BackendType, keySize int, valueSize int, dbPath string, ctx context.Context) []Step {
	initialStorageSize := 5 * units.GiB

	db, err := dbm.NewDB("experiment_db", backendType, dbPath)
	if err != nil {
		panic(err)
	}
	defer func(db dbm.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(db)

	var steps []Step
	initialRecordings := initialStorageSize / (keySize + valueSize)
	steps = append(steps, step("insert", initialRecordings, db, keySize, valueSize, dbPath, ctx))

	nFluctuations := 10
	recordingsPerStep := 1 * units.GiB / (keySize + valueSize)
	for i := 0; i < nFluctuations; i++ {
		select {
		case <-ctx.Done():
			return steps
		default:
			if i%2 == 0 {
				steps = append(steps, step("delete", recordingsPerStep, db, keySize, valueSize, dbPath, ctx))
			} else {
				steps = append(steps, step("insert", recordingsPerStep, db, keySize, valueSize, dbPath, ctx))
			}
		}
	}
	return steps
}

func batchedFluctuations() {

}

func fluctuationsForceCompact() {

}
