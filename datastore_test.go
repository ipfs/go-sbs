package fsbs

import (
	"os"
	"testing"

	dtest "github.com/ipfs/go-datastore/test"
)

func TestDatastoreBatch(t *testing.T) {
	dir := fsbsDir(t)
	fsds, err := NewSbsDS(dir)
	if err != nil {
		t.Fatal(err)
	}

	dtest.RunBatchTest(t, fsds)

	os.RemoveAll(dir)
}

func TestDatastoreBatchDelete(t *testing.T) {
	dir := fsbsDir(t)
	fsds, err := NewSbsDS(dir)
	if err != nil {
		t.Fatal(err)
	}

	dtest.RunBatchDeleteTest(t, fsds)

	os.RemoveAll(dir)
}

func TestDatastoreQuery(t *testing.T) {
	dir := fsbsDir(t)
	fsds, err := NewSbsDS(dir)
	if err != nil {
		t.Fatal(err)
	}

	dtest.SubtestManyKeysAndQuery(t, fsds)

	os.RemoveAll(dir)
}

func TestDatastorePutGet(t *testing.T) {
	dir := fsbsDir(t)
	fsds, err := NewSbsDS(dir)
	if err != nil {
		t.Fatal(err)
	}

	dtest.SubtestBasicPutGet(t, fsds)

	os.RemoveAll(dir)
}

func TestDatastoreNotFound(t *testing.T) {
	dir := fsbsDir(t)
	fsds, err := NewSbsDS(dir)
	if err != nil {
		t.Fatal(err)
	}

	dtest.SubtestNotFounds(t, fsds)

	os.RemoveAll(dir)
}
