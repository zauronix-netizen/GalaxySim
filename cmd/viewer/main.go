package main

import (
	"encoding/json"
	"fmt"
	"time"

	"galaxysim/sim"
)

type Snapshot struct {
	Day     int              `json:"day"`
	Star    int              `json:"star"`
	Objects []SnapshotObject `json:"objects"`
}

type SnapshotObject struct {
	T int     `json:"t"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func main() {
	fmt.Println("viewer: пытаюсь подключиться к shared memory...")
	shm, err := sim.OpenSharedMemReader()
	if err != nil {
		fmt.Printf("viewer: ошибка shared memory: %v\n", err)
		fmt.Println("убедись, что galaxysim.exe запущен и создал shared memory")
		return
	}
	defer shm.Close()

	fmt.Println("viewer: подключён. polling 500ms.")
	lastDay := -1
	for {
		raw := shm.Read()
		if len(raw) > 0 {
			var snap Snapshot
			if err := json.Unmarshal(raw, &snap); err != nil {
				fmt.Printf("viewer: bad json: %v\n", err)
			} else if snap.Day != lastDay {
				lastDay = snap.Day
				printSnapshot(snap)
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func printSnapshot(snap Snapshot) {
	planets, ships := 0, 0
	for _, o := range snap.Objects {
		switch o.T {
		case 1:
			planets++
		case 2:
			ships++
		}
	}
	fmt.Printf("\nday=%d star=%d | planets=%d ships=%d\n", snap.Day, snap.Star, planets, ships)
	for _, o := range snap.Objects {
		kind := "?"
		switch o.T {
		case 1:
			kind = "planet"
		case 2:
			kind = "ship"
		}
		fmt.Printf("  %-6s (%7.3f, %7.3f)\n", kind, o.X, o.Y)
	}
}