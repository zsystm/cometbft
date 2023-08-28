package db_experiments

import (
	dbm "github.com/cometbft/cometbft-db"
	"github.com/docker/go-units"
)

func inserts(backendType dbm.BackendType, keySize int, valueSize int, dbPath string) []Step {
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
		Name:    "initial",
		Size:    dirSize(dbPath),
		Records: dbCount(db),
	})

	currentStorageSize := 0
	recordingsPerStep := 1 * units.GiB / (keySize + valueSize)
	for currentStorageSize < targetStorageSize {
		steps = append(steps, step("insert", recordingsPerStep, db, keySize, valueSize, dbPath))
		currentStorageSize += (keySize + valueSize) * recordingsPerStep
	}

	return steps
}

func deletions(backendType dbm.BackendType, keySize int, valueSize int, dbPath string) []Step {
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
	steps = append(steps, step("insert", initialRecordings, db, keySize, valueSize, dbPath))

	recordingsPerStep := 1 * units.GiB / (keySize + valueSize)
	currentStorageSize := (keySize + valueSize) * initialRecordings
	for currentStorageSize > 0 {
		steps = append(steps, step("delete", recordingsPerStep, db, keySize, valueSize, dbPath))
		currentStorageSize -= (keySize + valueSize) * recordingsPerStep
	}

	return steps
}
func smallBatchInserts() {

}

func mediumBatchInserts() {

}

func bigBatchInserts() {

}

func smallFluctuations() {

}

func mediumFluctuations() {

}

func bigFluctuations() {

}

func smallBatchedFluctuations() {

}

func mediumBatchedFluctuations() {

}

func bigBatchedFluctuations() {

}

func smallFluctuationsForceCompact() {

}

func mediumFluctuationsForceCompact() {

}

func bigFluctuationsForceCompact() {

}

func smallGrowthWithDrops() {

}

func mediumGrowthWithDrops() {

}

func bigGrowthWithDrops() {

}
