package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "modernc.org/sqlite"
)

func databaseLocation() string {
	envOverride := os.Getenv("HS9001_DB_PATH")
	if envOverride != "" {
		return envOverride
	}
	return filepath.Join(xdgOrFallback("XDG_DATA_HOME", filepath.Join(os.Getenv("HOME"), ".local/share")), "hs9001/db.sqlite")
}

func createConnection() *sql.DB {

	db, err := sql.Open("sqlite", databaseLocation())
	if err != nil {
		log.Panic(err)
	}
	return db
}

func initDatabase(conn *sql.DB) {
	queryStmt := "CREATE TABLE history(id INTEGER PRIMARY KEY, command varchar(512), timestamp datetime DEFAULT current_timestamp, user varchar(25), hostname varchar(32));\n" +
		"CREATE VIEW count_by_date AS SELECT COUNT(id), STRFTIME('%Y-%m-%d', timestamp)  FROM history GROUP BY strftime('%Y-%m-%d', timestamp)"

	_, err := conn.Exec(queryStmt)
	if err != nil {
		log.Panic(err)
	}
}

func importFromStdin(conn *sql.DB) {
	scanner := bufio.NewScanner(os.Stdin)

	_, err := conn.Exec("BEGIN;")
	if err != nil {
		log.Panic(err)
	}

	for scanner.Scan() {
		add(conn, scanner.Text())
	}

	_, err = conn.Exec("END;")
	if err != nil {
		log.Panic(err)
	}
}

func search(conn *sql.DB, q string) {
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

func delete(conn *sql.DB, q string) {
	queryStmt := "DELETE FROM history WHERE command LIKE ?"

	_, err := conn.Exec(queryStmt, "%"+q+"%")
	if err != nil {
		log.Panic(err)
	}

	_, err = conn.Exec("VACUUM")
	if err != nil {
		log.Panic(err)
	}

}

func add(conn *sql.DB, cmd string) {
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

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:   ./hs9001 <add/search/init/import>\n")
}

func main() {
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)

	if len(os.Args) < 1 {
		printUsage()
		return
	}

	cmd := os.Args[1]
	globalargs := os.Args[2:]

	conn := createConnection()

	switch cmd {
	case "add":
		var ret int
		addCmd.IntVar(&ret, "ret", 0, "Return value of the command to add")
		addCmd.Parse(globalargs)
		args := addCmd.Args()

		if ret == 23 { // 23 is our secret do not log status code
			return
		}
		if len(args) < 1 {
			fmt.Fprint(os.Stderr, "Error: You need to provide the command to be added")

		}
		historycmd := args[0]
		var rgx = regexp.MustCompile("\\s+\\d+\\s+(.*)")
		rs := rgx.FindStringSubmatch(historycmd)
		if len(rs) == 2 {
			add(conn, rs[1])
		}
	case "search":
		searchCmd.Parse(globalargs)
		args := searchCmd.Args()

		if len(args) < 1 {
			fmt.Fprint(os.Stderr, "Please provide the search query\n")
			os.Exit(1)
		}

		q := strings.Join(args, " ")
		search(conn, q)
		os.Exit(23)
	case "delete":
		deleteCmd.Parse(globalargs)
		args := deleteCmd.Args()
		if len(args) < 1 {
			fmt.Fprint(os.Stderr, "Error: You need to provide a search query for records to delete")

		}
		q := strings.Join(args, " ")
		delete(conn, q)

		//we do not want to leak what we just deleted :^)
		os.Exit(23)
	case "init":
		err := os.MkdirAll(filepath.Dir(databaseLocation()), 0755)
		if err != nil {
			log.Panic(err)
		}
		initDatabase(conn)
	case "import":
		importFromStdin(conn)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown subcommand '%s' supplied\n\n", cmd)
		printUsage()
		return
	}

}
