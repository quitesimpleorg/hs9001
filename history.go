package main

import (
	"database/sql"
	"hs9001/liner"
	"io"
	"log"
	"path/filepath"
	"strings"
)

type history struct {
	conn *sql.DB
}

func createSearchOpts(query string, mode int) searchopts {
	opts := searchopts{}
	o := "DESC"
	opts.order = &o
	lim := 100
	opts.limit = &lim
	opts.command = &query

	switch mode {
	case liner.ModeGlobal:
		break
	case liner.ModeWorkdir:
		workdir, err := filepath.Abs(".")
		if err != nil {
			panic(err)
		}
		opts.workdir = &workdir
	default:
		panic("Invalid mode supplied")
	}
	return opts
}

func (h *history) GetHistoryByPrefix(prefix string, mode int) (ph []string) {
	cmdquery := prefix + "%"
	opts := createSearchOpts(cmdquery, mode)
	results := search(h.conn, opts)
	for e := results.Back(); e != nil; e = e.Prev() {
		entry, ok := e.Value.(*HistoryEntry)
		if !ok {
			log.Panic("Failed to retrieve entries")
		}
		ph = append(ph, entry.cmd)
	}
	return
}

func (h *history) GetHistoryByPattern(pattern string, mode int) (ph []string, pos []int) {
	cmdquery := "%" + pattern + "%"
	opts := createSearchOpts(cmdquery, mode)

	results := search(h.conn, opts)
	for e := results.Back(); e != nil; e = e.Prev() {
		entry, ok := e.Value.(*HistoryEntry)
		if !ok {
			log.Panic("Failed to retrieve entries")
		}
		ph = append(ph, entry.cmd)
		pos = append(pos, strings.Index(strings.ToLower(entry.cmd), strings.ToLower(pattern)))
	}
	return
}

func (h *history) ReadHistory(r io.Reader) (num int, err error) {
	panic("not implemented")
}
func (h *history) WriteHistory(w io.Writer) (num int, err error) {
	panic("not implemented")
}
func (h *history) AppendHistory(item string) {
	panic("not implemented")
}
func (h *history) ClearHistory() {
	panic("not implemented")
}
func (h *history) RLock() {
	//noop
}
func (h *history) RUnlock() {
	//noop
}
