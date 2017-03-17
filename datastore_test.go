package sbs

import (
	"os"
	"testing"

	dtest "github.com/ipfs/go-datastore/test"
)

func TestDatastoreBatch(t *testing.T) {
	dir := sbsDir(t)
	fsds, err := NewSbsDS(dir)
	if err != nil {
		t.Fatal(err)
	}

	dtest.RunBatchTest(t, fsds)

	os.RemoveAll(dir)
}

func TestDatastoreBatchDelete(t *testing.T) {
	dir := sbsDir(t)
	fsds, err := NewSbsDS(dir)
	if err != nil {
		t.Fatal(err)
	}

	dtest.RunBatchDeleteTest(t, fsds)

	os.RemoveAll(dir)
}

func TestDatastoreQuery(t *testing.T) {
	t.Skip("reenable after go-datastore update")
	dir := sbsDir(t)
	fsds, err := NewSbsDS(dir)
	if err != nil {
		t.Fatal(err)
	}

	_ = fsds
	//dtest.SubtestManyKeysAndQuery(t, fsds)

	os.RemoveAll(dir)
}

func TestDatastorePutGet(t *testing.T) {
	t.Skip("reenable after go-datastore update")
	dir := sbsDir(t)
	fsds, err := NewSbsDS(dir)
	if err != nil {
		t.Fatal(err)
	}

	_ = fsds
	//dtest.SubtestBasicPutGet(t, fsds)

	os.RemoveAll(dir)
}

func TestDatastoreNotFound(t *testing.T) {
	t.Skip("reenable after go-datastore update")
	dir := sbsDir(t)
	fsds, err := NewSbsDS(dir)
	if err != nil {
		t.Fatal(err)
	}

	_ = fsds
	//dtest.SubtestNotFounds(t, fsds)

	os.RemoveAll(dir)
}
