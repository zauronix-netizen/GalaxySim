package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type CaravanHistoryEntry struct {
	SourceID      int     `json:"sourceId"`
	DestinationID int     `json:"destId"`
	Good          int     `json:"good"`
	BuyPrice      float64 `json:"buyPrice"`
	SellPrice     float64 `json:"sellPrice"`
	FineTrade     bool    `json:"fineTrade"`
	Reason        string  `json:"reason"`
	Day           int     `json:"day"`
}

type CaravanJSON struct {
	ID          int                 `json:"id"`
	Race        int                 `json:"race"`
	TechLevel   int                 `json:"techLevel"`
	FromPlanet  int                 `json:"fromPlanet"`
	ToPlanet    int                 `json:"toPlanet"`
	Phase       int                 `json:"phase"`
	Good        int                 `json:"good"`
	Cargo       int                 `json:"cargo"`
	CargoMax    int                 `json:"cargoMax"`
	Money       float64             `json:"money"`
	Fuel        float64             `json:"fuel"`
	History     []CaravanHistoryEntry `json:"history"`
	HistoryLen  int                 `json:"historyLen"`
}

const baseURL = "http://localhost:8080"

var httpClient = &http.Client{Timeout: 5 * time.Second}

func fetchCaravan(id int) (*CaravanJSON, error) {
	url := fmt.Sprintf("%s/caravan/%d", baseURL, id)
	r, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", r.StatusCode)
	}
	var out CaravanJSON
	if err := json.NewDecoder(r.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func main() {
	id := flag.Int("id", -1, "Caravan ID")
	list := flag.Bool("list", false, "List all alive caravans (requires /world endpoint)")
	flag.String("url", baseURL, "Base URL (default: http://localhost:8080)")
	flag.Parse()

	if *list {
		// Можно добавить запрос /world или /caravan для списка, но тут проще через /world
		fmt.Println("Use /world endpoint to get total count, then query individual IDs")
		return
	}

	if *id < 0 {
		fmt.Println("Usage: caravan-history -id <caravan_id>")
		fmt.Println("       caravan-history -list (not implemented)")
		os.Exit(1)
	}

	caravan, err := fetchCaravan(*id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Caravan #%d\n", caravan.ID)
	fmt.Printf("  Race: %d, TechLevel: %d\n", caravan.Race, caravan.TechLevel)
	fmt.Printf("  From: %d -> To: %d, Phase: %d\n", caravan.FromPlanet, caravan.ToPlanet, caravan.Phase)
	fmt.Printf("  Good: %d, Cargo: %d/%d, Money: %.2f, Fuel: %.2f\n", caravan.Good, caravan.Cargo, caravan.CargoMax, caravan.Money, caravan.Fuel)
	fmt.Printf("  History: %d entries\n", caravan.HistoryLen)
	fmt.Println()

	if len(caravan.History) == 0 {
		fmt.Println("  (no history)")
		return
	}

	// Header
	fmt.Printf("%-6s %-10s %-10s %-5s %-10s %-10s %-6s %-12s %s\n", "DAY", "SRC", "DST", "GOOD", "BUY", "SELL", "OK", "REASON", "NOTE")
	fmt.Println(strings.Repeat("-", 92))

	for _, e := range caravan.History {
		reason := e.Reason
		if reason == "" {
			reason = "-"
		}
		okStr := "NO"
		if e.FineTrade {
			okStr = "YES"
		}
		note := ""
		if e.FineTrade && (e.BuyPrice > 0 || e.SellPrice > 0) {
			if e.BuyPrice > 0 && e.SellPrice == 0 {
				note = "BUY"
			} else if e.BuyPrice == 0 && e.SellPrice > 0 {
				note = "SELL"
			} else if e.BuyPrice > 0 && e.SellPrice > 0 {
				note = "ROUND"
			}
		}
		fmt.Printf("%-6d %-10d %-10d %-5d %-10.2f %-10.2f %-6s %-12s %s\n",
			e.Day, e.SourceID, e.DestinationID, e.Good, e.BuyPrice, e.SellPrice, okStr, reason, note)
	}
}