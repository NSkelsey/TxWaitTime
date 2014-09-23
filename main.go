package main

import (
	"bytes"
	"database/sql"
	"log"
	"math/rand"
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

var logger *log.Logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Llongfile)
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

	// Let the db handler open threads at will. This is to handle mainnet volume.
	db.SetMaxIdleConns(150)

	err = db.Ping()
	dieWith(err)

	rpcchan := make(chan *ResConn)

	go rpcroutine(client, rpcchan)

	// give the rpcroutine time to get some data
	time.Sleep(1)

	txParser := func(txmeta *watchtower.TxMeta) {
		txroutine(rpcchan, txmeta)
	}

	blockParser := func(now time.Time, block *btcwire.MsgBlock) {
		_hash, _ := block.BlockSha()
		hash := _hash.Bytes()
		logger.Printf("Saw block %v", hash)
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
		StartHeight: int(height),
		Net:         netparams.Net,
		Addr:        addr,
	}

	// Pass in closures and let them work
	watchtower.Create(cfg, txParser, blockParser)
}

func rpcroutine(client *btcrpcclient.Client, rpcchan <-chan *ResConn) {

	// This variable is a big unknown :-/
	var mempoolfut btcrpcclient.FutureGetRawMempoolVerboseResult

	tick := time.Tick(200 * time.Millisecond)
	chanmap := make(map[string]chan btcjson.GetRawMempoolResult)
	mempoolfut = client.GetRawMempoolVerboseAsync()
	for {
		var resconn *ResConn
		var txmempool map[string]btcjson.GetRawMempoolResult
		// The rpcroutine attempts to provide each txroutine with additional data
		// reported from an external data source
		select {
		case <-tick:
			// timeout if the rpc request takes too long
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

func handleSingleTx(rpcchan chan *ResConn, txmeta *watchtower.TxMeta) {
	// Tries to gather additional information about a tx and
	// commit the tx to the database.

	// change behavior if our tx is in a block
	var txInBlock bool = txmeta.BlockSha != nil
	txid, _ := txmeta.MsgTx.TxSha()
	txidbytes := txid.Bytes()

	// Extracting Tx Features
	counts := btcbuilder.ExtractOutScripts(txmeta.MsgTx)
	kind := btcbuilder.SelectKind(txmeta.MsgTx)
	size := txmeta.MsgTx.SerializeSize()

	// Copy tx bytes into a byte array for storage in the db
	buf := bytes.NewBuffer(make([]byte, 0, txmeta.MsgTx.SerializeSize()))
	txmeta.MsgTx.Serialize(buf)
	txbytes := buf.Bytes()
	dbTx, err := db.Begin()
	if err != nil {
		logger.Println(err)
		return
	}

	// Store Tx Features
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
	_, err = featStmt.Exec(txidbytes, counts[NonStandard], counts[PubKey], counts[PubKeyHash],
		counts[ScriptHash], counts[MultiSig], counts[NullData])
	if err != nil {
		logger.Println(err)
	}

	// Store the tx itself
	if txInBlock {
		// The Tx is in a block, therefore we should just store
		upStmt, _ := dbTx.Prepare(`
			INSERT INTO txs(txid, kind, firstseen, size, raw)
			SELECT $1, $2, $3, $4, $5
			WHERE NOT EXISTS (
				SELECT * FROM txs WHERE txid=$1
			);
		`)
		_, err = upStmt.Exec(txidbytes, kind, txmeta.Time, size, txbytes)
		if err != nil {
			logger.Println(err)
		}

	} else {
		// The Tx is not in a block, therefore we can collect extra info about it.

		// We use a crazy RPC call to get info from bitcoins mempool
		jsonChan := make(chan btcjson.GetRawMempoolResult, 1)
		rpcchan <- &ResConn{
			Txid: txid.String(),
			Chan: jsonChan,
		}

		var extra bool = false
		var fee int64 = 0
		var priority float64 = 0

		timeout := time.NewTimer(time.Millisecond * 250)
		var mempooljson btcjson.GetRawMempoolResult
		select {
		case <-timeout.C:
			logger.Printf("Rpc timeout on tx: %v", txid)
		case mempooljson = <-jsonChan:
			amnt, _ := btcutil.NewAmount(mempooljson.Fee)
			// The goods!
			extra = true
			fee = int64(amnt)
			priority = mempooljson.StartingPriority
		}

		// Commit tx
		upStmt, _ := dbTx.Prepare(`
			INSERT INTO txs(txid, kind, firstseen, size, extra, fee, priority, raw)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8
			WHERE NOT EXISTS (
				SELECT * FROM txs WHERE txid=$1
			);
		`)
		_, err = upStmt.Exec(txidbytes, kind, txmeta.Time, size, extra, fee, priority, txbytes)
		if err != nil {
			logger.Println(err)
		}
	}

	// Store txouts
	txOutStmt, _ := dbTx.Prepare(`
		INSERT INTO txouts(txid, vout, val, kind)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (
			SELECT * FROM txouts where txid=$1 AND vout=$2
		);
	`)

	for vout, txout := range txmeta.MsgTx.TxOut {
		class := btcscript.GetScriptClass(txout.PkScript)
		_, err = txOutStmt.Exec(txidbytes, vout, txout.Value, class.String())
		if err != nil {
			logger.Println(err)
		}
	}

	if txInBlock {
		// Record that the tx is in a block
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
	logger.Printf("Commit %v", txid)
}

func txroutine(rpcchan chan *ResConn, txmeta *watchtower.TxMeta) {
	if txmeta.BlockSha != nil {
		// If tx is in a block, randomly sleep routine for interval between 0-1 minute
		backoff := time.Duration(rand.Intn(1000))
		time.Sleep(time.Millisecond * backoff)
	}
	handleSingleTx(rpcchan, txmeta)
}
