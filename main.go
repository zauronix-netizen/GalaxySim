package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"galaxysim/api"
	"galaxysim/sim"
)

func tickInterval(day int) time.Duration {
	const daysInYear = 576
	if day >= 10*daysInYear { // год >= 3310
		return 500 * time.Millisecond
	}
	return 250 * time.Millisecond
}

func currentTickMs(w *sim.World) time.Duration {
	if w.TickOverrideMs > 0 {
		return time.Duration(w.TickOverrideMs) * time.Millisecond
	}
	return tickInterval(w.Day)
}

func main() {
	sim.EnableANSI()
	fmt.Print("\033[2J\033[H") // очистить экран (Win10 ANSI)

	// ... дальше как было
	seed := flag.Int64("seed", 0, "world seed (0 = random)")
	//
	viewerStar := flag.Int("viewer-star", -1, "star ID для shared memory (-1 = случайная)")
	flag.Parse()
	if *seed == 0 {
		*seed = time.Now().UnixNano()
	}
	log.Printf("seed=%d", *seed)
	w := sim.NewWorld(*seed)

	// Выбор звезды для снимка: -1 → случайная, иначе указанная.
	if *viewerStar >= 0 && *viewerStar < len(w.Stars) {
		w.ViewStar = *viewerStar
	} else {
		w.ViewStar = int(time.Now().UnixNano() % int64(len(w.Stars)))
		if w.ViewStar < 0 {
			w.ViewStar = -w.ViewStar
		}
	}

	// Shared Memory для viewer.exe.
	shm, err := sim.OpenSharedMemWriter()
	if err != nil {
		log.Printf("shared mem: %v (viewer отключён)", err)
	} else {
		w.Shm = shm
		log.Printf("shared mem: OK, star=%d", w.ViewStar)
	}

	defer func() {
		if w.Shm != nil {
			w.Shm.Close()
		}
	}()

	w.SpawnInitialAgents()
	
	habitableCount := 0
	for _, p := range w.Planets {
		if p.Habitable {
			habitableCount++
		}
	}
	
	log.Printf("world: %d stars, %d planets (%d habitable)",
		len(w.Stars), len(w.Planets), habitableCount)

	// Fast-forward: пропускаем первые 300 тиков (прогрев) без логов.
	// Экономика «стартует» сразу, не ждём 10 минут.
	for i := 0; i < sim.GalaxyWarmupTurns; i++ {
		w.SimTick()
	}
	log.Printf("fast-forward: day=%d", w.Day)

	srv := &api.Server{World: w}
	mux := http.NewServeMux()
	srv.Register(mux)
	
	log.Println("HTTP on :8080")
	go func() {
		
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatal(err)
		}
	}()

	tick := time.NewTicker(currentTickMs(w))
	defer tick.Stop()

	w.SimTickTimed()
	lastMs := currentTickMs(w)
	for range tick.C {
		w.SimTickTimed()
		ms := currentTickMs(w)
		if ms != lastMs {
			tick.Reset(ms)
			lastMs = ms
			log.Printf("tick → %s (day=%d)", ms, w.Day)
		}
	}
}