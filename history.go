package main

import (
	"database/sql"
	"io"
	"log"
	"strings"
)

type history struct {
	conn *sql.DB
}

func (h *history) GetHistoryByPrefix(prefix string) (ph []string) {
	opts := searchopts{}
	opts.order = "DESC"
	cmdqry := prefix + "%"
	opts.command = &cmdqry
	results := search(h.conn, opts)
	for e := results.Front(); e != nil; e = e.Next() {
		entry, ok := e.Value.(*HistoryEntry)
		if !ok {
			log.Panic("Failed to retrieve entries")
		}
		ph = append(ph, entry.cmd)
	}
	return
}
func (h *history) GetHistoryByPattern(pattern string) (ph []string, pos []int) {
	opts := searchopts{}
	opts.order = "DESC"
	cmdqry := "%" + pattern + "%"
	opts.command = &cmdqry
	results := search(h.conn, opts)
	for e := results.Front(); e != nil; e = e.Next() {
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
