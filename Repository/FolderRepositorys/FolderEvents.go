package FolderRepositorys

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	connection "web_backend/Model/Connection"
	"web_backend/Model/NewFolder"
	tbFolder "web_backend/Model/TbFolder"
)

// Global, in-memory state of every staging folder that is queued or being
// moved/deleted, broadcast to every open /status tab over GET
// /folders/events. Replaces the old per-task ProgressChannels: one queue,
// one worker, any number of listeners, and a snapshot on (re)connect so a
// reload or a second device picks up exactly where things are.
//
// ponytail: in-memory, correct only while the backend runs as a single
// instance (it does — one container). Multi-instance would need Redis
// pub/sub plus the claim stored in the DB.

// FolderState is one queued/processing folder as sent in the snapshot.
type FolderState struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Op      string  `json:"op"`     // "move" | "delete"
	Status  string  `json:"status"` // "queued" | "processing"
	Percent float64 `json:"percent"`
	seq     int
}

// FolderEvent is one SSE message: Event is the SSE event name ("item",
// "progress", "changed"), Data is marshalled as JSON.
type FolderEvent struct {
	Event string
	Data  any
}

// SkippedFolder is an id a claim refused, with the reason.
type SkippedFolder struct {
	ID     int    `json:"id"`
	Reason string `json:"reason"`
}

type folderJob struct {
	ID   int
	Name string
	Op   string
}

type folderHub struct {
	mu          sync.Mutex
	inflight    map[int]*FolderState
	subscribers map[chan FolderEvent]struct{}
	seq         int
}

var hub = &folderHub{
	inflight:    map[int]*FolderState{},
	subscribers: map[chan FolderEvent]struct{}{},
}

// ponytail: fixed-size queue; a claim beyond 10000 waiting folders is
// refused rather than blocking the request.
var folderJobs = make(chan folderJob, 10000)

// claim atomically marks every id that isn't already in flight as queued.
// names holds the ids that exist in the folders table; anything else is
// skipped as not found. Exactly one caller ever wins a given id, which is
// what makes two users submitting overlapping selections safe.
func (h *folderHub) claim(ids []int, names map[int]string, op string) (claimed []folderJob, skipped []SkippedFolder) {
	h.mu.Lock()
	defer h.mu.Unlock()

	seen := map[int]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true

		name, ok := names[id]
		switch {
		case !ok:
			skipped = append(skipped, SkippedFolder{ID: id, Reason: "tidak ditemukan"})
		case h.inflight[id] != nil:
			skipped = append(skipped, SkippedFolder{ID: id, Reason: "sedang diproses"})
		default:
			h.seq++
			h.inflight[id] = &FolderState{ID: id, Name: name, Op: op, Status: "queued", seq: h.seq}
			claimed = append(claimed, folderJob{ID: id, Name: name, Op: op})
			h.broadcastLocked(FolderEvent{"item", map[string]any{"id": id, "name": name, "op": op, "status": "queued"}})
		}
	}
	return claimed, skipped
}

// release drops an id from the in-flight set (finished, failed, or never
// actually queued) and broadcasts its final status.
func (h *folderHub) release(id int, data map[string]any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.inflight, id)
	if data != nil {
		h.broadcastLocked(FolderEvent{"item", data})
	}
}

func (h *folderHub) setProcessing(id int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s := h.inflight[id]; s != nil {
		s.Status = "processing"
		h.broadcastLocked(FolderEvent{"item", map[string]any{"id": id, "name": s.Name, "op": s.Op, "status": "processing"}})
	}
}

func (h *folderHub) setPercent(id int, pct float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s := h.inflight[id]; s != nil {
		s.Percent = pct
		h.broadcastLocked(FolderEvent{"progress", map[string]any{"id": id, "percent": pct}})
	}
}

// broadcastLocked never blocks the caller: a subscriber whose buffer is
// full is dropped (channel closed). Its browser reconnects on its own
// (EventSource retry) and gets a fresh snapshot, so nothing is silently
// lost — the client just resyncs.
func (h *folderHub) broadcastLocked(evt FolderEvent) {
	for ch := range h.subscribers {
		select {
		case ch <- evt:
		default:
			delete(h.subscribers, ch)
			close(ch)
		}
	}
}

func (h *folderHub) subscribe() ([]FolderState, chan FolderEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	snapshot := make([]FolderState, 0, len(h.inflight))
	for _, s := range h.inflight {
		snapshot = append(snapshot, *s)
	}
	sort.Slice(snapshot, func(i, j int) bool { return snapshot[i].seq < snapshot[j].seq })

	ch := make(chan FolderEvent, 256)
	h.subscribers[ch] = struct{}{}
	return snapshot, ch
}

func (h *folderHub) unsubscribe(ch chan FolderEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subscribers[ch]; ok {
		delete(h.subscribers, ch)
		close(ch)
	}
}

// SubscribeFolderEvents returns the current queued/processing folders (in
// queue order) plus a channel of live events. Snapshot and registration
// happen under one lock, so no event can slip in between them. The channel
// is closed if the subscriber falls too far behind; call
// UnsubscribeFolderEvents when the connection ends.
func SubscribeFolderEvents() ([]FolderState, chan FolderEvent) { return hub.subscribe() }

func UnsubscribeFolderEvents(ch chan FolderEvent) { hub.unsubscribe(ch) }

// PublishFoldersChanged tells every open /status tab to re-fetch its page
// (e.g. after /update inserted or removed rows).
func PublishFoldersChanged() {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	hub.broadcastLocked(FolderEvent{"changed", map[string]any{}})
}

// EnqueueFolders claims ids for op ("move" | "delete") and puts the claimed
// ones on the single global queue, processed in order by StartFolderWorker.
func EnqueueFolders(ids []int, op string) (claimed []int, skipped []SkippedFolder, err error) {
	var rows []tbFolder.Folder
	if len(ids) > 0 {
		if err := connection.DB.Select("id", "name").Where("id IN ?", ids).Find(&rows).Error; err != nil {
			return nil, nil, err
		}
	}
	names := make(map[int]string, len(rows))
	for _, r := range rows {
		names[int(r.ID)] = r.Name
	}

	jobs, skipped := hub.claim(ids, names, op)
	claimed = []int{} // [] rather than null in the JSON response
	if skipped == nil {
		skipped = []SkippedFolder{}
	}
	for _, job := range jobs {
		select {
		case folderJobs <- job:
			claimed = append(claimed, job.ID)
		default:
			hub.release(job.ID, map[string]any{"id": job.ID, "op": op, "status": "failed", "error": "antrian penuh"})
			skipped = append(skipped, SkippedFolder{ID: job.ID, Reason: "antrian penuh"})
		}
	}
	return claimed, skipped, nil
}

// StartFolderWorker runs the single goroutine that drains the queue. One
// worker on purpose: every job copies to/removes from the same disk, so
// parallel jobs only fight over I/O, and a single order makes "queued"
// mean the same thing on every device.
func StartFolderWorker() {
	go func() {
		for job := range folderJobs {
			hub.setProcessing(job.ID)
			var err error
			if job.Op == "delete" {
				err = deleteOne(job.ID)
			} else {
				err = moveOne(job.ID)
			}

			result := map[string]any{"id": job.ID, "name": job.Name, "op": job.Op}
			switch {
			case err != nil:
				log.Printf("folder %s id=%d failed: %v", job.Op, job.ID, err)
				result["status"] = "failed"
				result["error"] = err.Error()
			case job.Op == "delete":
				result["status"] = "deleted"
			default:
				result["status"] = "moved"
			}
			hub.release(job.ID, result)
		}
	}()
}

// moveOne copies one staging folder to DST_DIR, then (only if the copy
// fully succeeded) records it in new_folders, drops its folders row in one
// small transaction that commits immediately, and finally removes the
// source directory. A failed copy leaves the source and DB untouched.
func moveOne(id int) error {
	var row tbFolder.Folder
	if err := connection.DB.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil // already moved/deleted elsewhere — nothing left to do
		}
		return err
	}
	if err := checkFolderName(row.Name); err != nil {
		return err
	}

	source := filepath.Join(os.Getenv("SRC_DIR"), row.Name)
	destination := filepath.Join(os.Getenv("DST_DIR"), row.Name)

	total := countFiles(source)
	done := 0
	lastSent := time.Time{}
	onFileDone := func() {
		done++
		// Throttled: a folder of many small files would otherwise flood
		// every open tab with hundreds of events per second.
		if total > 0 && (done == total || time.Since(lastSent) > 250*time.Millisecond) {
			hub.setPercent(id, min(100, float64(done)/float64(total)*100))
			lastSent = time.Now()
		}
	}

	if err := copyPaste(source, destination, onFileDone); err != nil {
		return err
	}

	newRow := NewFolder.NewFolder{
		Name:        row.Name,
		Thumbnail:   strings.Replace(row.Thumbnail, "/sementara/", "/new/", 1),
		IsCompleted: false,
		CreateAt:    time.Now(),
	}
	if err := connection.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&newRow).Error; err != nil {
			return err
		}
		return tx.Delete(&tbFolder.Folder{}, row.ID).Error
	}); err != nil {
		return err
	}

	// DB already reflects the move; a leftover source dir is only clutter
	// (the next /update would re-list it), so don't fail the item over it.
	if err := os.RemoveAll(source); err != nil {
		log.Printf("moved id=%d but could not remove source %s: %v", id, source, err)
	}
	return nil
}

// deleteOne removes one staging folder from disk and then its folders row.
// The row is kept if the directory can't be removed, so it stays visible
// and can be retried.
func deleteOne(id int) error {
	var row tbFolder.Folder
	if err := connection.DB.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	if err := checkFolderName(row.Name); err != nil {
		return err
	}

	if err := os.RemoveAll(filepath.Join(os.Getenv("SRC_DIR"), row.Name)); err != nil {
		return err
	}
	return connection.DB.Delete(&tbFolder.Folder{}, row.ID).Error
}

// checkFolderName guards the RemoveAll calls above: an empty name would
// make the path SRC_DIR itself, and a name with a separator or ".." could
// point outside it.
func checkFolderName(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("nama folder tidak valid: %q", name)
	}
	return nil
}

func countFiles(root string) int {
	n := 0
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			n++
		}
		return nil
	})
	return n
}
