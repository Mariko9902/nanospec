package validator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCheckpointProcessesPreserveEveryLane(t *testing.T) {
	directory := writeCycleFixture(t, map[string][]string{
		"story-01-order": {}, "story-02-payment": {}, "story-03-refund": {},
	})
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	barrier := filepath.Join(t.TempDir(), "start")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	commands := make([]*exec.Cmd, 0, 3)
	outputs := make([]strings.Builder, 3)
	ids := []string{"story-01-order", "story-02-payment", "story-03-refund"}
	for index, id := range ids {
		command := exec.CommandContext(ctx, executable, "-test.run=^TestCheckpointProcess$")
		command.Env = append(os.Environ(), "NANOSPEC_CHECKPOINT_TARGET="+filepath.Join(directory, id), "NANOSPEC_CHECKPOINT_START="+barrier)
		command.Stdout = &outputs[index]
		command.Stderr = &outputs[index]
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, command)
	}
	writeFixtureFile(t, barrier, "start")
	for index, command := range commands {
		if err := command.Wait(); err != nil {
			t.Fatalf("checkpoint %s: %v\n%s", ids[index], err, outputs[index].String())
		}
	}
	snapshot, result := LoadCycleSnapshot(directory)
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("snapshot result = %#v", result)
	}
	if len(snapshot.ActiveLanes) != len(ids) {
		t.Fatalf("lanes = %#v", snapshot.ActiveLanes)
	}
	for _, id := range ids {
		if lane, found := storyLane(id, snapshot.ActiveLanes); !found || lane.Fields["Owner"] != "clerk" {
			t.Fatalf("missing checkpoint for %s", id)
		}
	}
}

func TestCheckpointProcess(t *testing.T) {
	target := os.Getenv("NANOSPEC_CHECKPOINT_TARGET")
	if target == "" {
		t.Skip("no execution target")
	}
	deadline := time.Now().Add(10 * time.Second)
	for !fileExists(os.Getenv("NANOSPEC_CHECKPOINT_START")) {
		if time.Now().After(deadline) {
			t.Fatal("start signal not received")
		}
		time.Sleep(5 * time.Millisecond)
	}
	_, result := SaveStoryCheckpoint(target, CheckpointRequest{
		Owner: "clerk", Status: "active", Next: "confirm the result", Situation: "record selected", At: time.Now(),
	})
	if result.InputError != "" || len(result.Errors) != 0 {
		t.Fatalf("checkpoint result = %#v", result)
	}
}

func TestConcurrentViewSavesRejectOneStaleUpdate(t *testing.T) {
	file := filepath.Join(t.TempDir(), "record.md")
	writeFixtureFile(t, file, "original")
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for _, value := range []string{"first", "second"} {
		group.Add(1)
		go func(value string) {
			defer group.Done()
			<-start
			results <- SaveViewUpdates([]ViewUpdate{{File: file, Before: "original", After: value}})
		}(value)
	}
	close(start)
	group.Wait()
	close(results)
	saved, rejected := 0, 0
	for err := range results {
		if err == nil {
			saved++
		} else if strings.Contains(err.Error(), "file changed during validation") {
			rejected++
		} else {
			t.Fatal(err)
		}
	}
	if saved != 1 || rejected != 1 {
		t.Fatalf("saved = %d, rejected = %d", saved, rejected)
	}
}

func TestWriteLocksResolveDirectoryAliases(t *testing.T) {
	directory := t.TempDir()
	alias := filepath.Join(t.TempDir(), "records")
	if err := os.Symlink(directory, alias); err != nil {
		t.Skip(err)
	}
	first, err := lockWriteDirectories([]string{directory})
	if err != nil {
		t.Fatal(err)
	}
	path := first.handles[0].Path()
	first.release()
	second, err := lockWriteDirectories([]string{alias})
	if err != nil {
		t.Fatal(err)
	}
	defer second.release()
	if second.handles[0].Path() != path {
		t.Fatal("directory aliases use different write locks")
	}
}
