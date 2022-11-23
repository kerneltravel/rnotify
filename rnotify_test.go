package rnotify_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kerneltravel/rnotify"
)

func TestWatchDirRecursive(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rnotify_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	if err := os.Mkdir(filepath.Join(tempDir, "foo"), 0777); err != nil {
		t.Fatal(err)
	}

	if tempDir, err = filepath.EvalSymlinks(tempDir); err != nil {
		t.Fatal(err)
	}

	watcher, _ := rnotify.NewWatcher()
	defer watcher.Close()

	done := make(chan bool)

	fileA := filepath.Join(tempDir, "a")
	dirB := filepath.Join(tempDir, "b")
	fileC := filepath.Join(tempDir, "foo", "c")
	dirD := filepath.Join(tempDir, "foo", "d")
	fileE := filepath.Join(tempDir, "foo", "d", "e")
	events := map[string]fsnotify.Op{
		fileA: fsnotify.Create,
		dirB:  fsnotify.Create,
		fileC: fsnotify.Create,
		dirD:  fsnotify.Create,
		fileE: fsnotify.Create,
		dirB:  fsnotify.Remove,
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if op, found := events[event.Name]; found && op == event.Op&op {
					delete(events, event.Name)
				}

				if len(events) == 0 {
					done <- true
				}
			case <-time.After(3 * time.Second):
				done <- true
			}
		}
	}()

	if err = watcher.Add(tempDir); err != nil {
		t.Fatal(err)
	}

	os.WriteFile(fileA, []byte{'a'}, 0600)
	os.Mkdir(dirB, 0777)
	os.WriteFile(fileC, []byte{'c'}, 0600)
	os.Mkdir(dirD, 0777)
	time.Sleep(100 * time.Millisecond)
	os.WriteFile(fileE, []byte{'e'}, 0600)
	os.RemoveAll(dirB)

	<-done

	if len(events) != 0 {
		t.Fatalf("%+v events didn't occur", events)
	}
}

func TestIgnore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rnotify_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	if err := os.Mkdir(filepath.Join(tempDir, "foo"), 0777); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir(filepath.Join(tempDir, ".git"), 0777); err != nil {
		t.Fatal(err)
	}

	if tempDir, err = filepath.EvalSymlinks(tempDir); err != nil {
		t.Fatal(err)
	}

	watcher, _ := rnotify.NewWatcher()
	watcher.Ignore([]string{".git"})
	defer watcher.Close()

	done := make(chan bool)

	fileA := filepath.Join(tempDir, "a")
	fileB := filepath.Join(tempDir, "foo", "b")
	fileC := filepath.Join(tempDir, ".git", "c")
	events := map[string]fsnotify.Op{
		fileA: fsnotify.Create,
		fileB: fsnotify.Create,
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if op, found := events[event.Name]; found && op == event.Op&op {
					delete(events, event.Name)
				} else if strings.Contains(event.Name, ".git") {
					fmt.Printf("unexpected %+v occurred\n", event)
					done <- true
				}

				if len(events) == 0 {
					done <- true
				}
			case <-time.After(3 * time.Second):
				done <- true
			}
		}
	}()

	if err = watcher.Add(tempDir); err != nil {
		t.Fatal(err)
	}

	os.WriteFile(fileC, []byte{'c'}, 0600)
	os.WriteFile(fileA, []byte{'a'}, 0600)
	os.WriteFile(fileB, []byte{'b'}, 0600)
	time.Sleep(100 * time.Millisecond)

	<-done

	if len(events) != 0 {
		t.Fatalf("%+v events didn't occur", events)
	}
}
