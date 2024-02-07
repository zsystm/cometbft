package v2

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/orderedcode"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
)

var logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))

// MigrateBlockStore migrates the block store database from version 1 to 2.
func MigrateBlockStore(db dbm.DB) error {
	logger.Info("migrating block metas")
	if err := migratePrefix(db, []byte("H:"), blockMetaKey); err != nil {
		return fmt.Errorf("migrate block metas: %w", err)
	}
	if err := deletePrefix(db, []byte("H:")); err != nil {
		return fmt.Errorf("delete old block metas: %w", err)
	}

	logger.Info("migrating block commits")
	if err := migratePrefix(db, []byte("C:"), blockCommitKey); err != nil {
		return fmt.Errorf("migrate block commits: %w", err)
	}
	if err := deletePrefix(db, []byte("C:")); err != nil {
		return fmt.Errorf("delete old block commits: %w", err)
	}

	logger.Info("migrating seen commits")
	if err := migratePrefix(db, []byte("SC:"), seenCommitKey); err != nil {
		return fmt.Errorf("migrate seen commits: %w", err)
	}
	if err := deletePrefix(db, []byte("SC:")); err != nil {
		return fmt.Errorf("delete old seen commits: %w", err)
	}

	logger.Info("migrating extended commits")
	if err := migratePrefix(db, []byte("EC:"), extCommitKey); err != nil {
		return fmt.Errorf("migrate extended commits: %w", err)
	}
	if err := deletePrefix(db, []byte("EC:")); err != nil {
		return fmt.Errorf("delete old extended commits: %w", err)
	}

	logger.Info("migrating block parts")
	it, err := dbm.IteratePrefix(db, []byte("P:"))
	if err != nil {
		panic(err)
	}
	defer it.Close()

	for ; it.Valid(); it.Next() {
		height, partIndex, err := parseHeightFromPartKey(it.Key())
		if err != nil {
			logger.Info("skipping invalid key", "key", it.Key(), "err", err)
			continue
		}
		if err = db.Set(blockPartKey(height, partIndex), it.Value()); err != nil {
			return fmt.Errorf("db.Set: %w", err)
		}
	}

	if err := it.Error(); err != nil {
		panic(err)
	}
	if err := deletePrefix(db, []byte("P:")); err != nil {
		return fmt.Errorf("delete old block parts: %w", err)
	}

	logger.Info("migrating block hashes")
	it, err = dbm.IteratePrefix(db, []byte("BH:"))
	if err != nil {
		panic(err)
	}
	defer it.Close()

	for ; it.Valid(); it.Next() {
		hash, err := parseBlockHash(it.Key())
		if err != nil {
			logger.Info("skipping invalid key", "key", it.Key(), "err", err)
			continue
		}
		if err = db.Set(blockHashKey(hash), it.Value()); err != nil {
			return fmt.Errorf("db.Set: %w", err)
		}
	}

	if err := it.Error(); err != nil {
		panic(err)
	}
	if err := deletePrefix(db, []byte("BH:")); err != nil {
		return fmt.Errorf("delete old block hashes: %w", err)
	}

	return nil
}

func deletePrefix(db dbm.DB, prefix []byte) error {
	it, err := dbm.IteratePrefix(db, prefix)
	if err != nil {
		panic(err)
	}
	defer it.Close()

	for ; it.Valid(); it.Next() {
		if err = db.Delete(it.Key()); err != nil {
			return fmt.Errorf("db.Delete: %w", err)
		}
	}

	if err := it.Error(); err != nil {
		panic(err)
	}

	return nil
}

// migratePrefix migrates all keys with the given prefix to the new key format
// defined by keyFn.
func migratePrefix(db dbm.DB, prefix []byte, keyFn func(int64) []byte) error {
	it, err := dbm.IteratePrefix(db, prefix)
	if err != nil {
		panic(err)
	}
	defer it.Close()

	for ; it.Valid(); it.Next() {
		height, err := parseHeight(it.Key())
		if err != nil {
			logger.Info("skipping invalid key", "key", it.Key(), "err", err)
			continue
		}
		if err = db.Set(keyFn(height), it.Value()); err != nil {
			return fmt.Errorf("db.Set: %w", err)
		}
	}

	if err := it.Error(); err != nil {
		panic(err)
	}

	return nil
}

// parseHeight parses the height from a key of the form "{suffix}:{height}".
func parseHeight(key []byte) (int64, error) {
	parts := strings.Split(string(key), ":")
	if len(parts) != 2 {
		return -1, fmt.Errorf("expected key to have 2 parts, got %d", len(parts))
	}
	return strconv.ParseInt(parts[1], 10, 64)
}

// parseHeightFromPartKey parses the height and part index from a key of the
// form "{suffix}:{height}:{partIndex}".
func parseHeightFromPartKey(key []byte) (int64, int64, error) {
	parts := strings.Split(string(key), ":")
	if len(parts) != 3 {
		return -1, 0, fmt.Errorf("expected key to have 3 parts, got %d", len(parts))
	}
	height, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return -1, 0, fmt.Errorf("error parsing height: %w", err)
	}
	partIndex, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return -1, 0, fmt.Errorf("error parsing part index: %w", err)
	}
	return height, partIndex, nil
}

// parseBlockHash parses the block hash from a key of the form "{suffix}:{hash}".
func parseBlockHash(key []byte) ([]byte, error) {
	parts := bytes.Split(key, []byte(":"))
	if len(parts) != 2 {
		return nil, fmt.Errorf("expected key to have 2 parts, got %d", len(parts))
	}
	return parts[1], nil
}

func MigrateStateDB(db dbm.DB) error {
	logger.Info("migrating validators")
	if err := migratePrefix(db, []byte("validatorsKey:"), validatorsKey); err != nil {
		return fmt.Errorf("migrate validators: %w", err)
	}
	if err := deletePrefix(db, []byte("validatorsKey:")); err != nil {
		return fmt.Errorf("delete old validators: %w", err)
	}

	logger.Info("migrating consensus params")
	if err := migratePrefix(db, []byte("consensusParamsKey:"), consensusParamsKey); err != nil {
		return fmt.Errorf("migrate consensus params: %w", err)
	}
	if err := deletePrefix(db, []byte("consensusParamsKey:")); err != nil {
		return fmt.Errorf("delete old consensus params: %w", err)
	}

	logger.Info("migrating ABCI responses")
	if err := migratePrefix(db, []byte("abciResponsesKey:"), abciResponsesKey); err != nil {
		return fmt.Errorf("migrate ABCI responses: %w", err)
	}
	if err := deletePrefix(db, []byte("abciResponsesKey:")); err != nil {
		return fmt.Errorf("delete old ABCI responses: %w", err)
	}

	return nil
}

func MigrateEvidenceDB(db dbm.DB) error {
	logger.Info("migrating committed evidence")
	it, err := dbm.IteratePrefix(db, []byte{byte(0x00)})
	if err != nil {
		panic(err)
	}
	defer it.Close()

	for ; it.Valid(); it.Next() {
		height, hash, err := parseEvidenceKey(it.Key())
		if err != nil {
			logger.Debug("not an evidence key", "key", it.Key(), "err", err)
			continue
		}
		key, err := orderedcode.Append(nil, prefixCommitted, height, hash)
		if err != nil {
			panic(err)
		}

		if err = db.Set(key, it.Value()); err != nil {
			return fmt.Errorf("db.Set: %w", err)
		}

		// for safety, don't use deletePrefix here.
		if err = db.Delete(it.Key()); err != nil {
			return fmt.Errorf("db.Delete: %w", err)
		}
	}

	if err := it.Error(); err != nil {
		panic(err)
	}

	logger.Info("migrating pending evidence")
	it, err = dbm.IteratePrefix(db, []byte{byte(0x01)})
	if err != nil {
		panic(err)
	}
	defer it.Close()

	for ; it.Valid(); it.Next() {
		height, hash, err := parseEvidenceKey(it.Key())
		if err != nil {
			logger.Debug("not an evidence key", "key", it.Key(), "err", err)
			continue
		}
		key, err := orderedcode.Append(nil, prefixPending, height, hash)
		if err != nil {
			panic(err)
		}
		if err = db.Set(key, it.Value()); err != nil {
			return fmt.Errorf("db.Set: %w", err)
		}
		// for safety, don't use deletePrefix here.
		if err = db.Delete(it.Key()); err != nil {
			return fmt.Errorf("db.Delete: %w", err)
		}
	}

	if err := it.Error(); err != nil {
		panic(err)
	}

	return nil
}

func MigrateLightClientDB(db dbm.DB) error {
	var err error

	logger.Info("migrating light blocks")
	it, err := dbm.IteratePrefix(db, []byte("lb/"))
	if err != nil {
		panic(err)
	}
	defer it.Close()
	var sizeK []byte
	for ; it.Valid(); it.Next() {
		chainID, height, ok := parseLbKey(it.Key())
		logger.Info("ChainID ", chainID)
		sizeK = sizeKey([]byte(chainID))
		if !ok {
			logger.Info("skipping invalid key", "key", it.Key())
			continue
		}
		key, err := orderedcode.Append(nil, chainID, prefixLightBlock, height)
		if err != nil {
			panic(err)
		}
		if err = db.Set(key, it.Value()); err != nil {
			return fmt.Errorf("db.Set: %w", err)
		}

		// // one per chainID
		// if _, ok := chainIDtoSizeMap[chainID]; !ok {
		// 	chainIDtoSizeMap[chainID] = 1
		// } else {
		// 	chainIDtoSizeMap[chainID]++
		// }
	}

	if err := deletePrefix(db, []byte("lb/")); err != nil {
		return fmt.Errorf("delete old light blocks: %w", err)
	}

	logger.Info("Getting old size")
	var size []byte
	if size, err = db.Get([]byte("size")); err != nil {
		return fmt.Errorf("error retrieveing old size key: %w ", err)
	} else {
		fmt.Println(string(size))
	}

	logger.Info("deleting old size key")
	if err := db.Delete([]byte("size")); err != nil {
		return fmt.Errorf("delete old size: %w", err)
	}

	if err = db.Set(sizeK, size); err != nil {
		return fmt.Errorf("db.Set: %w", err)
	}

	// for chainID, size := range chainIDtoSizeMap {
	// 	logger.Info("setting size for", "chainID", chainID, "size", size)

	// 	if err = db.Set(sizeKey([]byte(chainID)), marshalSize(size)); err != nil {
	// 		return fmt.Errorf("db.Set: %w", err)
	// 	}
	// }

	return nil
}

var keyPattern = regexp.MustCompile(`^(lb)/([^/]*)/([0-9]+)$`)

func parseKey(key []byte) (part string, chainID string, height int64, ok bool) {
	submatch := keyPattern.FindSubmatch(key)
	if submatch == nil {
		return "", "", 0, false
	}
	part = string(submatch[1])
	chainID = string(submatch[2])
	height, err := strconv.ParseInt(string(submatch[3]), 10, 64)
	if err != nil {
		return "", "", 0, false
	}
	ok = true // good!
	return
}

// parseLbKey parses the chainID and height from a key of the form "lb/{chainID}/{height}".
func parseLbKey(key []byte) (chainID string, height int64, ok bool) {
	var part string
	part, chainID, height, ok = parseKey(key)
	if part != "lb" {
		return "", 0, false
	}
	return
}

//---------------------------------- KEY ENCODING -----------------------------------------

const (
	// prefixes are unique across all tm db's.
	prefixBlockMeta   = int64(0)
	prefixBlockPart   = int64(1)
	prefixBlockCommit = int64(2)
	prefixSeenCommit  = int64(3)
	prefixExtCommit   = int64(4)
	prefixBlockHash   = int64(5)
)

func blockMetaKey(height int64) []byte {
	key, err := orderedcode.Append(nil, prefixBlockMeta, height)
	if err != nil {
		panic(err)
	}
	return key
}

func blockPartKey(height int64, partIndex int64) []byte {
	key, err := orderedcode.Append(nil, prefixBlockPart, height, partIndex)
	if err != nil {
		panic(err)
	}
	return key
}

func blockCommitKey(height int64) []byte {
	key, err := orderedcode.Append(nil, prefixBlockCommit, height)
	if err != nil {
		panic(err)
	}
	return key
}

func seenCommitKey(height int64) []byte {
	key, err := orderedcode.Append(nil, prefixSeenCommit, height)
	if err != nil {
		panic(err)
	}
	return key
}

func extCommitKey(height int64) []byte {
	key, err := orderedcode.Append(nil, prefixExtCommit, height)
	if err != nil {
		panic(err)
	}
	return key
}

func blockHashKey(hash []byte) []byte {
	key, err := orderedcode.Append(nil, prefixBlockHash, hash)
	if err != nil {
		panic(err)
	}
	return key
}

const (
	// prefixes must be unique across all db's.
	prefixValidators      = int64(6)
	prefixConsensusParams = int64(7)
	prefixABCIResponses   = int64(8)
)

func encodeKey(prefix, height int64) []byte {
	res, err := orderedcode.Append(nil, prefix, height)
	if err != nil {
		panic(err)
	}
	return res
}

func validatorsKey(height int64) []byte {
	return encodeKey(prefixValidators, height)
}

func consensusParamsKey(height int64) []byte {
	return encodeKey(prefixConsensusParams, height)
}

func abciResponsesKey(height int64) []byte {
	return encodeKey(prefixABCIResponses, height)
}

const (
	// prefixes must be unique across all db's.
	prefixCommitted = int64(9)
	prefixPending   = int64(10)
)

func parseEvidenceKey(key []byte) (int64, string, error) {
	parts := bytes.Split(key, []byte("/"))
	if len(parts) != 2 {
		return -1, "", fmt.Errorf("expected key to have 2 parts, got %d", len(parts))
	}

	decoded, err := hex.DecodeString(string(parts[0]))
	if err != nil {
		return 0, "", fmt.Errorf("error decoding height: %w", err)
	}
	var height int64
	buf := bytes.NewReader(decoded)
	err = binary.Read(buf, binary.BigEndian, &height)
	if err != nil {
		return 0, "", fmt.Errorf("error decoding height: %w", err)
	}
	return height, strings.ToLower(string(parts[1])), nil
}

const (
	// prefixes must be unique across all db's.
	prefixLightBlock = int64(11)
	prefixSize       = int64(12)
)

func sizeKey(prefix []byte) []byte {
	key, err := orderedcode.Append(nil, prefix, prefixSize)
	if err != nil {
		panic(err)
	}
	return key
}
