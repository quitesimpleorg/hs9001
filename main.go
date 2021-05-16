package main

import (
	"bufio"
	"container/list"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tj/go-naturaldate"
	_ "modernc.org/sqlite"
)

type HistoryEntry struct {
	id        uint32
	cmd       string
	cwd       string
	hostname  string
	user      string
	timestamp time.Time
}

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

func migrateDatabase(conn *sql.DB, currentVersion int) {

	migrations := []string{
		"ALTER TABLE history add column workdir varchar(4096) DEFAULT ''",
	}

	if !(len(migrations) > currentVersion) {
		return
	}

	_, err := conn.Exec("BEGIN;")
	if err != nil {
		log.Panic(err)
	}
	for _, m := range migrations[currentVersion:] {
		_, err := conn.Exec(m)
		if err != nil {
			log.Panic(err)
		}

	}

	setDBVersion(conn, len(migrations))

	_, err = conn.Exec("END;")
	if err != nil {
		log.Panic(err)
	}
}

func fetchDBVersion(conn *sql.DB) int {
	rows, err := conn.Query("PRAGMA user_version;")
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	rows.Next()
	var res int
	rows.Scan(&res)
	return res
}

func setDBVersion(conn *sql.DB, ver int) {
	_, err := conn.Exec(fmt.Sprintf("PRAGMA user_version=%d", ver))
	if err != nil {
		log.Panic(err)
	}
}

func NewHistoryEntry(cmd string) HistoryEntry {
	wd, err := os.Getwd()
	if err != nil {
		log.Panic(err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		log.Panic(err)
	}
	return HistoryEntry{
		user:      os.Getenv("USER"),
		hostname:  hostname,
		cmd:       cmd,
		cwd:       wd,
		timestamp: time.Now(),
	}
}

func importFromStdin(conn *sql.DB) {
	scanner := bufio.NewScanner(os.Stdin)

	_, err := conn.Exec("BEGIN;")
	if err != nil {
		log.Panic(err)
	}

	for scanner.Scan() {
		entry := NewHistoryEntry(scanner.Text())
		entry.cwd = ""
		entry.timestamp = time.Unix(0, 0)
		add(conn, entry)
	}

	_, err = conn.Exec("END;")
	if err != nil {
		log.Panic(err)
	}
}

func search(conn *sql.DB, q string, workdir string, beginTime time.Time, endTime time.Time) list.List {
	queryStmt := "SELECT id, command, workdir, user, hostname FROM history WHERE timestamp BETWEEN datetime(?, 'unixepoch') AND datetime(?, 'unixepoch') AND command LIKE ? AND workdir LIKE ? ORDER BY timestamp ASC"

	rows, err := conn.Query(queryStmt, beginTime.Unix(), endTime.Unix(), q, workdir)
	if err != nil {
		log.Panic(err)
	}

	var result list.List
	defer rows.Close()
	for rows.Next() {
		var entry HistoryEntry
		err = rows.Scan(&entry.id, &entry.cmd, &entry.cwd, &entry.user, &entry.hostname)
		if err != nil {
			log.Panic(err)
		}
		result.PushBack(&entry)
	}
	return result
}

func delete(conn *sql.DB, entryId uint32) {
	queryStmt := "DELETE FROM history WHERE id = ?"

	_, err := conn.Exec(queryStmt, entryId)
	if err != nil {
		log.Panic(err)
	}
}

func add(conn *sql.DB, entry HistoryEntry) {
	stmt, err := conn.Prepare("INSERT INTO history (user, command, hostname, workdir, timestamp) VALUES (?, ?, ?, ?, datetime(?, 'unixepoch'))")
	if err != nil {
		log.Panic(err)
	}

	_, err = stmt.Exec(entry.user, entry.cmd, entry.hostname, entry.cwd, entry.timestamp.Unix())
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
	fmt.Fprintf(os.Stderr, "Usage:   ./hs9001 <add/search/import>\n")
}

func main() {
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	cmd := os.Args[1]
	globalargs := os.Args[2:]

	var conn *sql.DB
	ok, _ := exists(databaseLocation())

	if !ok {
		err := os.MkdirAll(filepath.Dir(databaseLocation()), 0755)
		if err != nil {
			log.Panic(err)
		}
		conn = createConnection()
		initDatabase(conn)
	} else {
		conn = createConnection()
	}

	migrateDatabase(conn, fetchDBVersion(conn))

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
			add(conn, NewHistoryEntry(rs[1]))
		}
	case "search":
		fallthrough
	case "delete":
		var workDir string
		var beginTime string
		var endTime string
		var distinct bool = true

		searchCmd.StringVar(&workDir, "workdir", "%", "Search only within this workdir")
		searchCmd.StringVar(&beginTime, "begin", "50 years ago", "Start searching from this timeframe")
		searchCmd.StringVar(&endTime, "end", "now", "End searching from this timeframe")
		searchCmd.BoolVar(&distinct, "distinct", true, "Remove consecutive duplicate commands from output")

		searchCmd.Parse(globalargs)

		args := searchCmd.Args()

		beginTimestamp, err := naturaldate.Parse(beginTime, time.Now())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to convert time string: %s\n", err.Error())
		}

		endTimeStamp, err := naturaldate.Parse(endTime, time.Now())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to convert time string: %s\n", err.Error())
		}

		q := strings.Join(args, " ")
		results := search(conn, "%"+q+"%", workDir, beginTimestamp, endTimeStamp)

		previousCmd := ""
		for e := results.Front(); e != nil; e = e.Next() {
			entry, ok := e.Value.(*HistoryEntry)
			if !ok {
				log.Panic("Failed to retrieve entries")
			}
			if !distinct || previousCmd != entry.cmd {
				fmt.Printf("%s\n", entry.cmd)
			}
			previousCmd = entry.cmd
		}

		if cmd == "delete" {

			_, err := conn.Exec("BEGIN;")
			if err != nil {
				log.Panic(err)
			}

			for e := results.Front(); e != nil; e = e.Next() {
				entry, ok := e.Value.(*HistoryEntry)
				if !ok {
					log.Panic("Failed to retrieve entries")
				}
				delete(conn, entry.id)
			}

			_, err = conn.Exec("END;")
			if err != nil {
				log.Panic(err)
			}

			_, err = conn.Exec("VACUUM")
			if err != nil {
				log.Panic(err)
			}

		}
		os.Exit(23)
	case "import":
		importFromStdin(conn)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown subcommand '%s' supplied\n\n", cmd)
		printUsage()
		return
	}

}
