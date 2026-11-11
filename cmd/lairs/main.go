package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type SysPlanet struct {
	ID        int     `json:"id"`
	Orbit     float64 `json:"orbit"`
	Habitable bool    `json:"habitable"`
	Race      int     `json:"race"`
}
type Sys struct {
	StarID  int         `json:"starId"`
	Planets []SysPlanet `json:"planets"`
}

type Lair struct {
	ID        int     `json:"id"`
	StarID    int     `json:"starId"`
	Orbit     float64 `json:"orbit"`
	Race      int     `json:"race"`
	Money     float64 `json:"money"`
	TechLevel int     `json:"techLevel"`
	Cargo     []int   `json:"cargo"`
}

var base = "http://localhost:8080"

func getJSON(url string, v interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func findLairs() []int {
	ids := []int{}
	for i := 0; i < 600; i++ {
		var s Sys
		if err := getJSON(fmt.Sprintf("%s/system/%d", base, i), &s); err != nil {
			continue
		}
		for _, p := range s.Planets {
			if !p.Habitable && p.Orbit >= 4.2 && p.Orbit <= 5.8 {
				ids = append(ids, p.ID)
				break
			}
		}
	}
	return ids
}

func printLairs(ids []int) {
	for i, id := range ids {
		fmt.Printf("%6d", id)
		if (i+1)%16 == 0 {
			fmt.Println()
		}
	}
	if len(ids)%16 != 0 {
		fmt.Println()
	}
	fmt.Printf("lairs: %d\n", len(ids))
}

func fetchLair(id int) (*Lair, error) {
	var l Lair
	err := getJSON(fmt.Sprintf("%s/lair/%d", base, id), &l)
	return &l, err
}

func printLair(l *Lair) {
	fmt.Printf("planet: %d\n", l.ID)
	fmt.Printf("star:   %d\n", l.StarID)
	fmt.Printf("orbit:  %.2f\n", l.Orbit)
	fmt.Printf("race:   %d\n", l.Race)
	fmt.Printf("tech:   %d\n", l.TechLevel)
	fmt.Printf("money:  %.4g\n", l.Money)
	fmt.Println("cargo:")
	for i, c := range l.Cargo {
		fmt.Printf("  %d: %d\n", i+1, c)
	}
}

func printStats(ids []int) {
	n := len(ids)
	if n == 0 {
		fmt.Println("no lairs")
		return
	}
	var totalMoney float64
	sums := make([]int, 8)
	for _, id := range ids {
		l, err := fetchLair(id)
		if err != nil {
			fmt.Printf("planet %d: error %v\n", id, err)
			continue
		}
		totalMoney += l.Money
		for i := 0; i < 8 && i < len(l.Cargo); i++ {
			sums[i] += l.Cargo[i]
		}
	}
	fmt.Printf("lairs: %d\n", n)
	fmt.Printf("avg money: %.4g\n", totalMoney/float64(n))
	fmt.Println("avg cargo:")
	for i := 0; i < 8; i++ {
		fmt.Printf("  %d: %.2f\n", i+1, float64(sums[i])/float64(n))
	}
}

func main() {
	if len(os.Args) > 1 {
		base = os.Args[1]
	}
	ids := findLairs()
	printLairs(ids)

	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !sc.Scan() {
			return
		}
		in := strings.TrimSpace(sc.Text())
		if in == "" {
			printLairs(ids)
			continue
		}
		if in == "-1" {
			return
		}
		if in == "0" {
			printStats(ids)
			continue
		}
		n, err := strconv.Atoi(in)
		if err != nil {
			fmt.Println("error input")
			printLairs(ids)
			continue
		}
		found := false
		for _, id := range ids {
			if id == n {
				found = true
				break
			}
		}
		if !found {
			fmt.Println("error input")
			printLairs(ids)
			continue
		}
		l, err := fetchLair(n)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			continue
		}
		printLair(l)
	}
}