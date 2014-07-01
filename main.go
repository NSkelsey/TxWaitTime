package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/NSkelsey/btcbuilder"
	"github.com/NSkelsey/watchtower"
	"github.com/conformal/btcjson"
	"github.com/conformal/btcrpcclient"
	"github.com/conformal/btcscript"
	"github.com/conformal/btcutil"
	"github.com/conformal/btcwire"
	"github.com/lib/pq"
)

var logger *log.Logger = log.New(os.Stdout, "", log.Ltime|log.Ldate)
var db *sql.DB

// some golang ugliness
var NonStandard, PubKey, PubKeyHash, ScriptHash, MultiSig, NullData btcscript.ScriptClass = btcscript.NonStandardTy, btcscript.PubKeyTy, btcscript.PubKeyHashTy,
	btcscript.ScriptHashTy, btcscript.MultiSigTy, btcscript.NullDataTy

func dieWith(err error) {
	if err != nil {
		logger.Fatal(err)
	}
}

type ResConn struct {
	Txid string
	Chan chan btcjson.GetRawMempoolResult
}

func main() {
	client, netparams := btcbuilder.ConfigureApp()
	addr := "127.0.0.1:" + netparams.DefaultPort

	connurl := "postgres://postgres:obscureref@localhost/txwaittime"
	var err error
	db, err = sql.Open("postgres", connurl)
	defer db.Close()
	dieWith(err)

	err = db.Ping()
	dieWith(err)

	rpcchan := make(chan *ResConn)

	go rpcroutine(client, rpcchan)

	// give the rpcroutine time to get some data
	time.Sleep(1)

	txParser := func(txmeta *watchtower.TxMeta) {

		go txroutine(rpcchan, txmeta)
	}

	blockParser := func(now time.Time, block *btcwire.MsgBlock) {
		logger.Println("Saw block")
		_hash, _ := block.BlockSha()
		hash := _hash.Bytes()
		// insert block
		_, err := db.Exec(`INSERT INTO blocks(hash, firstseen) VALUES($1, $2)`,
			hash, now)
		if err != nil {
			logger.Println(err)
		}

		if err, ok := err.(*pq.Error); ok {
			logger.Println("pq error:", err.Code.Name())
		}
		if err != nil {
			logger.Println(err)
		}
	}

	height, err := client.GetBlockCount()
	dieWith(err)

	cfg := watchtower.TowerCfg{
		StartHeight: height,
		Net:         netparams.Net,
		Addr:        addr,
	}

	// Pass in closures and let them work
	watchtower.Create(cfg, netparams.Net, txParser, blockParser)
}

func rpcroutine(client *btcrpcclient.Client, rpcchan <-chan *ResConn) {

	// This variable is a big unknown :-/
	var mempoolfut btcrpcclient.FutureGetRawMempoolVerboseResult

	tick := time.Tick(500 * time.Millisecond)
	chanmap := make(map[string]chan btcjson.GetRawMempoolResult)
	mempoolfut = client.GetRawMempoolVerboseAsync()
	for {
		var resconn *ResConn
		var txmempool map[string]btcjson.GetRawMempoolResult
		// The rpcroutine attempts to provide each txroutine with additional data
		// reported from an external data source
		select {
		case <-tick:
			//logger.Println("ticked for rpc")
			// try to recieve from future
			var err error
			txmempool, err = mempoolfut.Receive()
			if err != nil {
				logger.Println(err)
				break
			}

			for txid, json := range txmempool {
				if txchan, ok := chanmap[txid]; ok {
					txchan <- json
					close(txchan)
					delete(chanmap, txid)
				}

			}
			mempoolfut = client.GetRawMempoolVerboseAsync()

		case resconn = <-rpcchan:
			// receive from one of the channels in
			chanmap[resconn.Txid] = resconn.Chan
		}
	}
}

func txroutine(rpcchan chan *ResConn, txmeta *watchtower.TxMeta) {
	var now time.Time
	if txmeta.BlockSha != nil {
		now = txmeta.Time
	} else {
		now = time.Now()
	}
	txid, err := txmeta.MsgTx.TxSha()
	if err != nil {
		logger.Println(err)
		return
	}
	counts := btcbuilder.ExtractOutScripts(txmeta.MsgTx)
	kind := btcbuilder.SelectKind(txmeta.MsgTx)
	size := txmeta.MsgTx.SerializeSize()

	jsonChan := make(chan btcjson.GetRawMempoolResult, 1)

	rpcchan <- &ResConn{
		Txid: txid.String(),
		Chan: jsonChan,
	}

	var extra bool
	var fee int64
	var priority float64

	timeout := time.NewTimer(time.Second * 1)
	var mempooljson btcjson.GetRawMempoolResult
	select {
	case <-timeout.C:
		// insert with null values
		extra = false
		fee = 0
		priority = 0

	case mempooljson = <-jsonChan:
		amnt, _ := btcutil.NewAmount(mempooljson.Fee)
		logger.Println(amnt)
		// The goods!
		extra = true
		fee = int64(amnt)
		priority = mempooljson.StartingPriority

	}
	txbytes := txid.Bytes()
	dbTx, err := db.Begin()
	if err != nil {
		logger.Println(err)
		return
	}

	featStmt, err := dbTx.Prepare(`
	INSERT INTO tx_features(txid, nonstandard, pubkey, pubkeyhash, scripthash, multisig, nulldata)
	SELECT $1, $2, $3, $4, $5, $6, $7
	WHERE NOT EXISTS (
		SELECT * FROM tx_features WHERE txid=$1
	);
	`)
	if err != nil {
		logger.Println(err)
	}
	_, err = featStmt.Exec(txbytes, counts[NonStandard], counts[PubKey], counts[PubKeyHash],
		counts[ScriptHash], counts[MultiSig], counts[NullData])
	if err != nil {
		logger.Println(err)
	}

	upStmt, _ := dbTx.Prepare(`
	INSERT INTO txs(txid, kind, firstseen, size, extra, fee, priority)
	SELECT $1, $2, $3, $4, $5, $6, $7
	WHERE NOT EXISTS (
		SELECT * FROM txs WHERE txid=$1
	);
	`)
	_, err = upStmt.Exec(txbytes, kind, now, size, extra, fee, priority)
	if err != nil {
		logger.Println(err)
	}

	if txmeta.BlockSha != nil {
		_, err = dbTx.Exec(`INSERT INTO relations(txid, block) VALUES($1, $2)`,
			txid.Bytes(), txmeta.BlockSha)
		if err != nil {
			logger.Println(err)
		}
	}

	err = dbTx.Commit()
	if err != nil {
		logger.Println(err)
	}
}
