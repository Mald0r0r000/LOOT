package hash

import (
	"io/ioutil"
	"os"
	"testing"

	"loot/internal/config"
)

func TestHasher_Single(t *testing.T) {
	input := []byte("test")

	// MD5 Test
	mh := NewHasher(config.AlgoMD5)
	mh.Write(input)
	res := mh.Sum()

	expectedMD5 := "098f6bcd4621d373cade4e832627b4f6"
	if res.MD5 != expectedMD5 {
		t.Errorf("MD5 mismatch. Got %s, want %s", res.MD5, expectedMD5)
	}

	// SHA256 Test
	mh2 := NewHasher(config.AlgoSHA256)
	mh2.Write(input)
	res2 := mh2.Sum()

	expectedSHA := "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
	if res2.SHA256 != expectedSHA {
		t.Errorf("SHA256 mismatch. Got %s, want %s", res2.SHA256, expectedSHA)
	}
}

func TestHasher_Multi(t *testing.T) {
	input := []byte("test")

	// Dual Hash Test (XXHash + MD5)
	mh := NewMultiHasher(config.AlgoXXHash64, config.AlgoMD5)
	mh.Write(input)
	res := mh.Sum()

	expectedMD5 := "098f6bcd4621d373cade4e832627b4f6"
	if res.MD5 != expectedMD5 {
		t.Errorf("MD5 mismatch in multi. Got %s", res.MD5)
	}
	if res.XXHash64 == "" {
		t.Error("XXHash64 is empty")
	}
}

func TestCalculateFileHash(t *testing.T) {
	// Create temp file
	tmpFile, err := ioutil.TempFile("", "loot_test_hash")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := []byte("test")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Calculate
	res, err := CalculateFileHash(tmpFile.Name(), config.AlgoMD5)
	if err != nil {
		t.Fatal(err)
	}

	expectedMD5 := "098f6bcd4621d373cade4e832627b4f6"
	if res.MD5 != expectedMD5 {
		t.Errorf("File hash mismatch. Got %s, want %s", res.MD5, expectedMD5)
	}
}
