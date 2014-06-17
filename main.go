package main

import (
	"database/sql"
	"log"
	"os"
	"time"
	"watchtower"

	"github.com/conformal/btcjson"
	"github.com/conformal/btcrpcclient"
	"github.com/conformal/btcutil"
	"github.com/conformal/btcwire"
	"github.com/lib/pq"
)

var logger *log.Logger = log.New(os.Stdout, "", log.Ltime|log.Llongfile)
var connurl string = "postgres://postgres:obscureref@localhost/txwaittime"

func handle(err error) {
	if err != nil {
		logger.Fatal(err)
	}
}

func selectKind(tx *btcwire.MsgTx) string {
	return "p2sh"
}

type ResConn struct {
	Txid string
	Chan chan btcjson.GetRawMempoolResult
}

func pinger(conn *sql.DB) {
	for {
		time.Sleep(1 * time.Second)
		handle(conn.Ping())
		logger.Println("tick")
	}
}

func main() {

	tConn, err := sql.Open("postgres", connurl)
	handle(err)
	bConn, err := sql.Open("postgres", connurl)
	handle(err)

	go pinger(tConn)

	rpcchan := make(chan *ResConn)

	go rpcroutine(rpcchan)

	// give the rpcroutine time to get some data
	time.Sleep(1)

	txParser := func(txmeta *watchtower.TxMeta) {
		logger.Println("Saw tx")

		go txroutine(rpcchan, txmeta)
	}

	blockParser := func(block *btcwire.MsgBlock) {
		logger.Println("Saw block")
		_hash, _ := block.BlockSha()
		hash := _hash.Bytes()
		// insert block
		_, err := bConn.Query(`INSERT INTO blocks(hash, firstseen) VALUES($1, $2)`,
			hash, time.Now())

		if err, ok := err.(*pq.Error); ok {
			logger.Println("pq error:", err.Code.Name())
		}
		if err != nil {
			logger.Println(err)
		}
	}

	// Pass in closures and let them work
	watchtower.Create(txParser, blockParser)
}

func rpcroutine(rpcchan <-chan *ResConn) {
	connCfg := btcrpcclient.ConnConfig{
		Host:         "localhost:18332",
		User:         "bitcoinrpc",
		Pass:         "EhxWGNKr1Z4LLqHtfwyQDemCRHF8gem843pnLj19K4go",
		HttpPostMode: true,
		DisableTLS:   true,
	}

	client, err := btcrpcclient.New(&connCfg, nil)
	if err != nil {
		logger.Fatal(err)
	}

	// fuck this variable :-/
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
			logger.Println("ticked for rpc")
			// try to recieve from future
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
	txid, _ := txmeta.MsgTx.TxSha()
	kind := selectKind(txmeta.MsgTx)
	now := time.Now()
	size := txmeta.MsgTx.SerializeSize()

	jsonChan := make(chan btcjson.GetRawMempoolResult, 1)

	rpcchan <- &ResConn{
		Txid: txid.String(),
		Chan: jsonChan,
	}

	conn, err := sql.Open("postgres", connurl)
	if err != nil {
		logger.Println(err)
		return
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
		// The goods!
		extra = true
		fee = int64(amnt)
		priority = mempooljson.StartingPriority

	}

	insert_new := `
	INSERT INTO txs(txid, kind, firstseen, size, extra, fee, priority) 
		SELECT $1, $2, $3, $4, $5, $6, $7
		WHERE NOT EXISTS (
			SELECT * FROM txs WHERE txid=$1
		);
	`
	_, err = conn.Exec(insert_new, txid.Bytes(), kind, now, size, extra, fee, priority)
	if err != nil {
		logger.Println(err)
		return
	}

	if txmeta.BlockSha != nil {
		_, err = conn.Exec(`INSERT INTO relations(txid, block) VALUES($1, $2)`,
			txid.Bytes(), txmeta.BlockSha)
		if err != nil {
			logger.Println(err)
			return
		}
	}
}
