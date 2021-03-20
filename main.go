package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	_ "github.com/mattn/go-sqlite3"
)

func create_connection() * sql.DB {
	homedir := os.Getenv("HOME");

	db, err := sql.Open("sqlite3", homedir + "./local/share/hs0001/db.sqlite")
	if err != nil {
		log.Panic(err)
	}
	return db
}

func search(q string) {
	conn := create_connection()

	queryStmt := "SELECT command FROM history WHERE command LIKE ? ORDER BY timestamp ASC"

	rows, err := conn.Query(queryStmt,"%"+q+"%");
	if err != nil {
		log.Panic(err)
	}

	for rows.Next() {
		var command string
		err = rows.Scan(&command)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("%s", command)
	}
}

func add(cmd string) {
	conn := create_connection()

	user := os.Getenv("USER")
	hostname, err := os.Hostname()
	if err != nil {
		log.Panic(err)
	}

	stmt, err := conn.Prepare("INSERT INTO history (user, command, hostname) VALUES (?, ?, ?)")
	if err != nil {
		log.Panic(err)
	}

	_, err = stmt.Exec(user, cmd, hostname)
	if err != nil {
		log.Panic(err)
	}

}


func main() {
	flag.Parse()
	args := flag.Args()
	argslen := len(args)
	cmd := args[0]

	if cmd == "add" {
		if argslen < 2 {
			fmt.Fprint(os.Stderr, "Error: You need to provide the command to be added")

		}
		historycmd := args[1]
		var rgx = regexp.MustCompile("\\s+\\d+\\s+(.*)")
		rs := rgx.FindStringSubmatch(historycmd)
		add(rs[1])
	} else if cmd == "search" {
		if argslen < 2 {
			fmt.Fprint(os.Stderr, "Please provide the search query\n")
		}
		q := args[1]
		search(q)
	}
}
