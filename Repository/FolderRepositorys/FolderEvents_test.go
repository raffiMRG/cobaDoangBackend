package FolderRepositorys

import (
	"reflect"
	"testing"
)

func newTestHub() *folderHub {
	return &folderHub{inflight: map[int]*FolderState{}, subscribers: map[chan FolderEvent]struct{}{}}
}

func jobIDs(jobs []folderJob) []int {
	ids := []int{}
	for _, j := range jobs {
		ids = append(ids, j.ID)
	}
	return ids
}

// User A picks 1,2,5 and user B picks 1,3,5 at the same time: every folder
// must be claimed exactly once, and B is told which ones it lost.
func TestClaimOverlappingUsers(t *testing.T) {
	h := newTestHub()
	names := map[int]string{1: "a", 2: "b", 3: "c", 5: "e"}

	a, aSkipped := h.claim([]int{1, 2, 5}, names, "move")
	b, bSkipped := h.claim([]int{1, 3, 5, 9}, names, "move")

	if got := jobIDs(a); !reflect.DeepEqual(got, []int{1, 2, 5}) || len(aSkipped) != 0 {
		t.Fatalf("A claimed %v skipped %v", got, aSkipped)
	}
	if got := jobIDs(b); !reflect.DeepEqual(got, []int{3}) {
		t.Fatalf("B claimed %v, want [3]", got)
	}
	want := []SkippedFolder{{1, "sedang diproses"}, {5, "sedang diproses"}, {9, "tidak ditemukan"}}
	if !reflect.DeepEqual(bSkipped, want) {
		t.Fatalf("B skipped %v, want %v", bSkipped, want)
	}

	// Once released (done or failed), the id can be claimed again.
	h.release(1, nil)
	if c, _ := h.claim([]int{1}, names, "delete"); len(c) != 1 {
		t.Fatalf("id 1 not claimable after release")
	}
}

// A new subscriber (reload / second device) gets every in-flight folder in
// queue order, then live events.
func TestSubscribeSnapshotThenLive(t *testing.T) {
	h := newTestHub()
	names := map[int]string{7: "x", 3: "y"}
	h.claim([]int{7, 3}, names, "move")
	h.setProcessing(7)

	snap, ch := h.subscribe()
	if len(snap) != 2 || snap[0].ID != 7 || snap[0].Status != "processing" || snap[1].ID != 3 || snap[1].Status != "queued" {
		t.Fatalf("snapshot %+v", snap)
	}

	h.release(7, map[string]any{"id": 7, "status": "moved"})
	if evt := <-ch; evt.Event != "item" || evt.Data.(map[string]any)["status"] != "moved" {
		t.Fatalf("live event %+v", evt)
	}
	if snap, _ := h.subscribe(); len(snap) != 1 || snap[0].ID != 3 {
		t.Fatalf("snapshot after release %+v", snap)
	}
}

// A subscriber that stops reading must never block the worker.
func TestSlowSubscriberDropped(t *testing.T) {
	h := newTestHub()
	_, ch := h.subscribe()
	names := map[int]string{}
	for i := 1; i <= 300; i++ {
		names[i] = "f"
		h.claim([]int{i}, names, "move")
	}
	n := 0
	for range ch { // closed once the buffer overflowed
		n++
	}
	if n == 0 || len(h.subscribers) != 0 {
		t.Fatalf("slow subscriber not dropped (got %d events, %d subscribers)", n, len(h.subscribers))
	}
}

func TestCheckFolderName(t *testing.T) {
	for _, bad := range []string{"", ".", "..", "a/b", `a\b`} {
		if checkFolderName(bad) == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if checkFolderName("One Piece v01") != nil {
		t.Error("normal name rejected")
	}
}
