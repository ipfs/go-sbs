package sbs

import (
	pb "github.com/ipfs/go-sbs/pb"

	proto "github.com/gogo/protobuf/proto"
	ds "github.com/ipfs/go-datastore"
	query "github.com/ipfs/go-datastore/query"
	"github.com/jbenet/goprocess"

	"github.com/boltdb/bolt"
)

func (fs *Sbsds) Query(q query.Query) (query.Results, error) {
	qrb := query.NewResultBuilder(q)

	qrb.Process.Go(func(worker goprocess.Process) {
		fs.sbs.index.View(func(tx *bolt.Tx) error {

			buck := tx.Bucket(bucketOffset)
			c := buck.Cursor()

			var prefix []byte
			if qrb.Query.Prefix != "" {
				prefix = []byte(qrb.Query.Prefix)
			}

			cur := 0
			sent := 0
			for k, v := c.Seek(prefix); k != nil; k, v = c.Next() {
				if cur < qrb.Query.Offset {
					cur++
					continue
				}
				if qrb.Query.Limit > 0 && sent >= qrb.Query.Limit {
					break
				}
				dk := ds.RawKey(string(k)).String()
				e := query.Entry{Key: dk}

				if !qrb.Query.KeysOnly {
					var prec pb.Record

					err := proto.Unmarshal(v, &prec)
					if err != nil {
						qrb.Output <- query.Result{Error: err}
						return err
					}
					l := prec.GetSize_()
					buf := make([]byte, l)
					fs.sbs.read(&prec, buf)

					e.Value = buf
				}

				select {
				case qrb.Output <- query.Result{Entry: e}: // we sent it out
					sent++
				case <-worker.Closing(): // client told us to end early.
					break
				}
				cur++
			}

			return nil
		})
	})

	// go wait on the worker (without signaling close)
	go qrb.Process.CloseAfterChildren()

	qr := qrb.Results()
	for _, f := range q.Filters {
		qr = query.NaiveFilter(qr, f)
	}
	for _, o := range q.Orders {
		qr = query.NaiveOrder(qr, o)
	}
	return qr, nil
}
