package db_experiments

import (
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/docker/go-units"
	"github.com/stretchr/testify/require"
)

func TestFillStorageSequential(t *testing.T) {
	db := dbm.NewMemDB()
	defer func(db dbm.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(db)

	fillStorageToVolumeSequentialKeys(5*units.MiB, 1*units.MiB, db)
	itr, err := db.Iterator(nil, nil)
	require.NoError(t, err)
	var keys []uint64
	for ; itr.Valid(); itr.Next() {
		keys = append(keys, bytesToUint64(itr.Key()))
	}
	require.Equal(t, []uint64{0, 1, 2, 3}, keys)
}
