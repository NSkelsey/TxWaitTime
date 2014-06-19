package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	connurl := "postgres://postgres:obscureref@localhost/txwaittime"
	db, err := sql.Open("postgres", connurl)
	if err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query(`SELECT * FROM txs;`)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("success!")
	for rows.Next() {
		var txid []byte
		var kind string
		var seen time.Time
		var extra bool
		var priority float64
		var size, fee int

		err := rows.Scan(&txid, &kind, &seen, &size, &extra, &priority, &fee)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("%x, %s\n", txid, kind)
	}
}
