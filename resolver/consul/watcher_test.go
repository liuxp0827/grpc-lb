package consul

import (
	"log"
	"testing"
)

func TestWatcher(t *testing.T) {
	watcher, err := NewWatcher("192.168.1.75:8500", "echo")
	if err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case addr := <-watcher.Watch():
			for _, a := range addr {
				log.Printf("address: %v", a)
			}
		}
	}
}
