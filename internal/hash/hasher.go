package hash

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"

	"loot/internal/config"

	"github.com/cespare/xxhash/v2"
)

// MultiHasher can calculate multiple hashes simultaneously
type MultiHasher struct {
	xxh     hash.Hash64
	md5Hash hash.Hash
	sha     hash.Hash

	enabledAlgos map[config.HashAlgorithm]bool
}

// NewMultiHasher creates a hasher for the given algorithms
func NewMultiHasher(algorithms ...config.HashAlgorithm) *MultiHasher {
	mh := &MultiHasher{
		enabledAlgos: make(map[config.HashAlgorithm]bool),
	}

	for _, algo := range algorithms {
		mh.enabledAlgos[algo] = true

		switch algo {
		case config.AlgoXXHash64:
			mh.xxh = xxhash.New()
		case config.AlgoMD5:
			mh.md5Hash = md5.New()
		case config.AlgoSHA256:
			mh.sha = sha256.New()
		}
	}

	return mh
}

// NewHasher creates a hasher for a single algorithm
func NewHasher(algo config.HashAlgorithm) *MultiHasher {
	return NewMultiHasher(algo)
}

// Write implements io.Writer
func (mh *MultiHasher) Write(p []byte) (n int, err error) {
	if mh.xxh != nil {
		mh.xxh.Write(p)
	}
	if mh.md5Hash != nil {
		mh.md5Hash.Write(p)
	}
	if mh.sha != nil {
		mh.sha.Write(p)
	}
	return len(p), nil
}

// HashResult contains all calculated hashes
type HashResult struct {
	XXHash64 string
	MD5      string
	SHA256   string
}

// Sum returns all calculated hashes
func (mh *MultiHasher) Sum() HashResult {
	result := HashResult{}

	if mh.xxh != nil {
		result.XXHash64 = fmt.Sprintf("%x", mh.xxh.Sum64())
	}
	if mh.md5Hash != nil {
		result.MD5 = hex.EncodeToString(mh.md5Hash.Sum(nil))
	}
	if mh.sha != nil {
		result.SHA256 = hex.EncodeToString(mh.sha.Sum(nil))
	}

	return result
}

// GetPrimary returns the hash for the primary algorithm
func (result HashResult) GetPrimary(algo config.HashAlgorithm) string {
	switch algo {
	case config.AlgoXXHash64:
		return result.XXHash64
	case config.AlgoMD5:
		return result.MD5
	case config.AlgoSHA256:
		return result.SHA256
	default:
		return ""
	}
}

// String returns a formatted string of all hashes
func (result HashResult) String() string {
	s := ""
	if result.XXHash64 != "" {
		s += fmt.Sprintf("xxhash64:%s ", result.XXHash64)
	}
	if result.MD5 != "" {
		s += fmt.Sprintf("md5:%s ", result.MD5)
	}
	if result.SHA256 != "" {
		s += fmt.Sprintf("sha256:%s", result.SHA256)
	}
	return s
}

// CalculateFileHash calculates hash(es) for a file
func CalculateFileHash(path string, algorithms ...config.HashAlgorithm) (HashResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return HashResult{}, err
	}
	defer f.Close()

	hasher := NewMultiHasher(algorithms...)

	// Use 4MB buffer for efficient reading
	buf := make([]byte, 4*1024*1024)
	if _, err := io.CopyBuffer(hasher, f, buf); err != nil {
		return HashResult{}, err
	}

	return hasher.Sum(), nil
}
