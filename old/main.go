package main

import (
	"log"
	"net/http"
	"time"

	"galaxysim/api"
	"galaxysim/sim"
)

func main() {
	w := sim.NewWorld(12345)
	w.SpawnInitialAgents()
	log.Printf("world: %d stars, %d planets, %d caravans, %d pirates, %d liners, %d diplomats",
		len(w.Stars), len(w.Planets), len(w.Caravans), len(w.Pirates), len(w.Liners), len(w.Diplomats))

	// Fast-forward: пропускаем первые 300 тиков (прогрев) без логов.
	// Экономика «стартует» сразу, не ждём 10 минут.
	for i := 0; i < 300; i++ {
		w.SimTick()
	}
	log.Printf("fast-forward: day=%d", w.Day)

	srv := &api.Server{World: w}
	mux := http.NewServeMux()
	srv.Register(mux)
	go func() {
		log.Println("HTTP on :8080")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatal(err)
		}
	}()

	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	w.SimTickTimed()
	for range tick.C {
		w.SimTickTimed()
	}
}