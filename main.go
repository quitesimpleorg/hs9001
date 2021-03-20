package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	_ "modernc.org/sqlite"
)

func databaseLocation() string {
	return filepath.Join(xdgOrFallback("XDG_DATA_HOME", filepath.Join(os.Getenv("HOME"), ".local/share")), "hs9001/db.sqlite")
}

func createConnection() *sql.DB {

	db, err := sql.Open("sqlite", databaseLocation())
	if err != nil {
		log.Panic(err)
	}
	return db
}

func initDatabase() {
	conn := createConnection()

	queryStmt := "CREATE TABLE history(id INTEGER PRIMARY KEY, command varchar(512), timestamp datetime DEFAULT current_timestamp, user varchar(25), hostname varchar(32));\n" +
		"CREATE VIEW count_by_date AS SELECT COUNT(id), STRFTIME('%Y-%m-%d', timestamp)  FROM history GROUP BY strftime('%Y-%m-%d', timestamp)"

	_, err := conn.Exec(queryStmt)
	if err != nil {
		log.Panic(err)
	}
}

func search(q string) {
	conn := createConnection()

	queryStmt := "SELECT command FROM history WHERE command LIKE ? ORDER BY timestamp ASC"

	rows, err := conn.Query(queryStmt, "%"+q+"%")
	if err != nil {
		log.Panic(err)
	}

	for rows.Next() {
		var command string
		err = rows.Scan(&command)
		if err != nil {
			log.Panic(err)
		}
		fmt.Printf("%s\n", command)
	}
}

func add(cmd string) {
	conn := createConnection()

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

func xdgOrFallback(xdg string, fallback string) string {
	dir := os.Getenv(xdg)
	if dir != "" {
		if ok, err := exists(dir); ok && err == nil {
			return dir
		}

	}

	return fallback
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func main() {
	flag.Parse()
	args := flag.Args()
	argslen := len(args)

	if argslen < 1 {
		fmt.Fprintf(os.Stderr, "Usage:   ./hs9001 <add/search/init>\n")
		return
	}

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
	} else if cmd == "init" {
		err := os.MkdirAll(filepath.Dir(databaseLocation()), 0755)
		if err != nil {
			log.Panic(err)
		}
		initDatabase()
	}
}
