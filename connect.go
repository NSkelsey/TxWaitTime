package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	connurl := "postgres://postgres:obscureref@localhost/txwaittime"
	db, err := sql.Open("postgres", connurl)
	if err != nil {
		log.Fatal(err)
	}

	r, err := db.Query(`SELECT * FROM txs;`)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(r)
}
