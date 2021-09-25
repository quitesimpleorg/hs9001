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
	"strconv"
	"strings"
	"time"

	"hs9001/liner"

	"github.com/tj/go-naturaldate"
	_ "modernc.org/sqlite"
)

type HistoryEntry struct {
	id        uint32
	cmd       string
	cwd       string
	hostname  string
	user      string
	retval    int
	timestamp time.Time
}

var GitTag string
var GitCommit string

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
		"ALTER TABLE history ADD COLUMN workdir varchar(4096) DEFAULT ''",
		"ALTER TABLE history ADD COLUMN retval integer DEFAULT -9001",
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

func NewHistoryEntry(cmd string, retval int) HistoryEntry {
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
		retval:    retval,
	}
}

func importFromStdin(conn *sql.DB) {
	scanner := bufio.NewScanner(os.Stdin)

	_, err := conn.Exec("BEGIN;")
	if err != nil {
		log.Panic(err)
	}

	for scanner.Scan() {
		entry := NewHistoryEntry(scanner.Text(), -9001)
		entry.cwd = ""
		entry.timestamp = time.Unix(0, 0)
		add(conn, entry)
	}

	_, err = conn.Exec("END;")
	if err != nil {
		log.Panic(err)
	}
}

type searchopts struct {
	command *string
	workdir *string
	after   *time.Time
	before  *time.Time
	retval  *int
	order   *string
	limit   *int
}

func search(conn *sql.DB, opts searchopts) list.List {
	args := make([]interface{}, 0)
	var sb strings.Builder
	sb.WriteString("SELECT id, command, workdir, user, hostname, retval, strftime(\"%s\", timestamp) ")
	sb.WriteString("FROM history ")
	sb.WriteString("WHERE 1=1 ") //1=1 so we can append as many AND foo as we want, or none

	if opts.command != nil {
		sb.WriteString("AND command LIKE ? ")
		args = append(args, opts.command)
	}
	if opts.workdir != nil {
		sb.WriteString("AND workdir LIKE ? ")
		args = append(args, opts.workdir)
	}
	if opts.after != nil {
		sb.WriteString("AND timestamp > datetime(?, 'unixepoch') ")
		args = append(args, opts.after.Unix())
	}
	if opts.before != nil {
		sb.WriteString("AND timestamp < datetime(?, 'unixepoch') ")
		args = append(args, opts.before.Unix())
	}
	if opts.retval != nil {
		sb.WriteString("AND retval = ? ")
		args = append(args, opts.retval)
	}
	sb.WriteString("ORDER BY timestamp ")
	if opts.order != nil {
		sb.WriteString(*opts.order)
		sb.WriteRune(' ')
	} else {
		sb.WriteString("ASC ")
	}

	if opts.limit != nil {
		sb.WriteString("LIMIT ")
		sb.WriteString(strconv.Itoa(*opts.limit))
		sb.WriteRune(' ')
	}

	queryStmt := sb.String()

	rows, err := conn.Query(queryStmt, args...)
	if err != nil {
		log.Panic(err)
	}

	var result list.List
	defer rows.Close()
	for rows.Next() {
		var entry HistoryEntry
		var timestamp int64
		err = rows.Scan(&entry.id, &entry.cmd, &entry.cwd, &entry.user, &entry.hostname, &entry.retval, &timestamp)
		if err != nil {
			log.Panic(err)
		}
		entry.timestamp = time.Unix(timestamp, 0)
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
	stmt, err := conn.Prepare("INSERT INTO history (user, command, hostname, workdir, timestamp, retval) VALUES (?, ?, ?, ?, datetime(?, 'unixepoch'),?)")
	if err != nil {
		log.Panic(err)
	}

	_, err = stmt.Exec(entry.user, entry.cmd, entry.hostname, entry.cwd, entry.timestamp.Unix(), entry.retval)
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
	fmt.Fprintf(os.Stderr, "Usage:   ./hs9001 <add/search/import/nolog/bash-enable>\n")
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
	case "bash-ctrlr":
		line := liner.NewLiner()
		defer line.Close()
		line.SetCtrlCAborts(true)
		line.SetHistoryProvider(&history{conn: conn})
		line.SetMultiLineMode(true)

		rdlineline := os.Getenv("READLINE_LINE")
		rdlinepos := os.Getenv("READLINE_POS")
		rdlineposint, _ := strconv.Atoi(rdlinepos)

		if name, err := line.PromptWithSuggestionReverse("", rdlineline, rdlineposint); err == nil {
			fmt.Fprintf(os.Stderr, "%s\n", name)
		}

	case "bash-enable":
		fmt.Printf(`
			if [ -n "$PS1" ] ; then
				PROMPT_COMMAND='hs9001 add -ret $? "$(history 1)"'
				bind -x '"\C-r": " READLINE_LINE=$(hs9001 bash-ctrlr 3>&1 1>&2 2>&3) READLINE_POINT=0"'
			fi
			alias hs='hs9001 search'
		`)
	case "bash-disable":
		fmt.Printf("unset PROMPT_COMMAND\n")
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
		var rgx = regexp.MustCompile(`\s+\d+\s+(.*)`)
		rs := rgx.FindStringSubmatch(historycmd)
		if len(rs) == 2 {
			add(conn, NewHistoryEntry(rs[1], ret))
		}
	case "search":
		fallthrough
	case "delete":
		var workDir string
		var afterTime string
		var beforeTime string
		var distinct bool = true
		var today bool = false
		var retVal int
		searchCmd.StringVar(&workDir, "cwd", "", "Search only within this workdir")
		searchCmd.StringVar(&afterTime, "after", "", "Start searching from this timeframe")
		searchCmd.StringVar(&beforeTime, "before", "", "End searching from this timeframe")
		searchCmd.BoolVar(&distinct, "distinct", true, "Remove consecutive duplicate commands from output")
		searchCmd.BoolVar(&today, "today", false, "Search only today's entries. Overrides --after")
		searchCmd.IntVar(&retVal, "ret", -9001, "Only query commands that returned with this exit code. -9001=all (default)")
		searchCmd.Parse(globalargs)

		args := searchCmd.Args()

		q := strings.Join(args, " ")

		opts := searchopts{}
		o := "ASC"
		opts.order = &o
		if q != "" {
			cmd := "%" + q + "%"
			opts.command = &cmd
		}
		if workDir != "" {
			wd, err := filepath.Abs(workDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed parse working directory path: %s\n", err.Error())
			}
			opts.workdir = &wd
		}

		if today {
			afterTime = "today"
		}

		if afterTime != "" {
			afterTimestamp, err := naturaldate.Parse(afterTime, time.Now())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to convert time string: %s\n", err.Error())
			}
			opts.after = &afterTimestamp
		}
		if beforeTime != "" {
			beforeTimestamp, err := naturaldate.Parse(beforeTime, time.Now())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to convert time string: %s\n", err.Error())
			}
			opts.before = &beforeTimestamp
		}
		if retVal != -9001 {
			opts.retval = &retVal
		}
		results := search(conn, opts)

		previousCmd := ""

		fi, err := os.Stdout.Stat()
		if err != nil {
			panic(err)
		}

		//Don't print colors if output is piped
		printColors := true
		if (fi.Mode() & os.ModeCharDevice) == 0 {
			printColors = false
		}

		for e := results.Front(); e != nil; e = e.Next() {
			entry, ok := e.Value.(*HistoryEntry)
			if !ok {
				log.Panic("Failed to retrieve entries")
			}
			if !distinct || previousCmd != entry.cmd {
				prefix := ""
				postfix := ""
				if printColors && entry.retval != 0 {
					prefix = "\033[38;5;88m"
					postfix = "\033[0m"
				}
				fmt.Printf("%s%s%s\n", prefix, entry.cmd, postfix)
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
	case "version":
		fmt.Fprintf(os.Stdout, "Git Tag: %s\nGit Commit: %s\n", GitTag, GitCommit)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown subcommand '%s' supplied\n\n", cmd)
		printUsage()
		return
	}

}
