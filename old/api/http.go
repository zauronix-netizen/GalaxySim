package api

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strconv"

	"galaxysim/sim"
)

//go:embed index.html
var indexHTML []byte

type Server struct {
	World *sim.World
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/planet/", s.handlePlanet)
	mux.HandleFunc("/map", s.handleMap)
	mux.HandleFunc("/world", s.handleWorld)
	mux.HandleFunc("/habitable", s.handleHabitable)
	mux.HandleFunc("/news", s.handleNews)
	mux.HandleFunc("/relations", s.handleRelations)
	mux.HandleFunc("/", s.handleIndex)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}

type MarketJSON struct {
	Good          int     `json:"good"`
	PriceState    float64 `json:"priceState"`
	PurchasePrice float64 `json:"purchasePrice"`
	BaseSalePrice float64 `json:"baseSalePrice"`
	Count         int     `json:"count"`
	TargetStock   int     `json:"targetStock"`
	TargetPrice   float64 `json:"targetPrice"`
	Base          int     `json:"base"`
	Legal         bool    `json:"legal"`
	Delta         float64 `json:"delta"`
}

type PlanetJSON struct {
	Day        int          `json:"day"`
	ID         int          `json:"id"`
	Star       int          `json:"star"`
	Sector     int          `json:"sector"`    // Star.Sector, 0..80
	TechLevel  int          `json:"techLevel"` // 0..15, hex 0x00..0x0F в вебе
	Race       int          `json:"race"`
	RaceName   string       `json:"raceName"`
	Government string       `json:"government"`
	Pop        int          `json:"pop"`
	Money      float64      `json:"money"`
	Economy    string       `json:"economy"`
	Habitable  bool         `json:"habitable"`
	Market     []MarketJSON `json:"market"`
}

type MapSeedJSON struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type MapStarJSON struct {
	ID     int     `json:"id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Sector int     `json:"sector"`
}
type MapEdgeJSON struct {
	A int `json:"a"`
	B int `json:"b"`
}
type MapJSON struct {
	MapW  float64       `json:"mapW"`
	MapH  float64       `json:"mapH"`
	Seeds []MapSeedJSON `json:"seeds"`
	Stars []MapStarJSON `json:"stars"`
	Edges []MapEdgeJSON `json:"edges"`
}

func (s *Server) handleMap(w http.ResponseWriter, r *http.Request) {
	out := MapJSON{
		MapW:  sim.MapW,
		MapH:  sim.MapH,
		Seeds: make([]MapSeedJSON, 0, len(s.World.SectorSeeds)),
		Stars: make([]MapStarJSON, 0, len(s.World.Stars)),
	}
	for _, sd := range s.World.SectorSeeds {
		out.Seeds = append(out.Seeds, MapSeedJSON{X: sd.X, Y: sd.Y})
	}
	for _, st := range s.World.Stars {
		out.Stars = append(out.Stars, MapStarJSON{ID: st.ID, X: st.X, Y: st.Y, Sector: st.Sector})
	}
	for _, edges := range s.World.ConstellationEdges {
		for _, e := range edges {
			out.Edges = append(out.Edges, MapEdgeJSON{A: e.A, B: e.B})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func raceName(r sim.Race) string {
	switch r {
	case sim.RaceMaloc:
		return "Maloc"
	case sim.RacePeleng:
		return "Peleng"
	case sim.RaceHuman:
		return "Human"
	case sim.RaceFeyan:
		return "Feyan"
	case sim.RaceGaal:
		return "Gaal"
	}
	return "?"
}

func economyShort(e sim.Economy) string {
	switch e {
	case sim.EcoAgri:
		return "agri"
	case sim.EcoMixed:
		return "mixed"
	case sim.EcoInd:
		return "ind"
	}
	return "?"
}

func governmentShort(g sim.Government) string {
	switch g {
	case sim.GovAnarchy:
		return "anar"
	case sim.GovDictatorship:
		return "dict"
	case sim.GovMonarchy:
		return "mon"
	case sim.GovRepublic:
		return "rep"
	case sim.GovDemocracy:
		return "dem"
	}
	return "?"
}

func governmentName(g sim.Government) string {
	switch g {
	case sim.GovAnarchy:
		return "anarchy"
	case sim.GovDictatorship:
		return "dictatorship"
	case sim.GovMonarchy:
		return "monarchy"
	case sim.GovRepublic:
		return "republic"
	case sim.GovDemocracy:
		return "democracy"
	}
	return "?"
}

func economyName(e sim.Economy) string {
	switch e {
	case sim.EcoAgri:
		return "agricultural"
	case sim.EcoMixed:
		return "mixed"
	case sim.EcoInd:
		return "industrial"
	}
	return "unknown"
}

func (s *Server) handlePlanet(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/planet/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= len(s.World.Planets) {
		http.Error(w, "bad planet id", http.StatusBadRequest)
		return
	}
	p := s.World.Planets[id]
	resp := PlanetJSON{
		Day:        s.World.Day,
		ID:         p.ID,
		Star:       p.StarID,
		Sector:     s.World.Stars[p.StarID].Sector,
		TechLevel:  p.TechLevel,
		Race:       int(p.Race),
		RaceName:   raceName(p.Race),
		Government: governmentShort(p.Government),
		Pop:        p.Pop,
		Money:      p.Money,
		Economy:    economyShort(p.Economy),
		Habitable:  p.Habitable,
		Market:     make([]MarketJSON, sim.GoodCount),
	}
	for g := 0; g < int(sim.GoodCount); g++ {
		m := &p.Market[g]
		resp.Market[g] = MarketJSON{
			Good:          g,
			PriceState:    m.PriceState,
			PurchasePrice: m.PurchasePrice,
			BaseSalePrice: m.BaseSalePrice,
			Count:         m.Count,
			TargetStock:   m.TargetStock,
			TargetPrice:   m.TargetPrice,
			Base:          p.BaseStock[g],
			Legal:         s.World.IsLegal(p, sim.Good(g)),
			Delta:         m.Delta,
		}
	}
	// Порядок вывода для веба: GoodsTextOrder (продукты, медикаменты, алкоголь, ...).
	ordered := make([]MarketJSON, 0, len(resp.Market))
	for _, idx := range sim.GoodsTextOrder {
		ordered = append(ordered, resp.Market[idx])
	}
	resp.Market = ordered
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

type WorldJSON struct {
	Day               int     `json:"day"`
	Stars             int     `json:"stars"`
	Planets           int     `json:"planets"`
	Caravans          int     `json:"caravans"`
	Pirates           int     `json:"pirates"`
	Liners            int     `json:"liners"`
	Diplomats         int     `json:"diplomats"`
	TotalPlanetMoney  float64 `json:"totalPlanetMoney"`
	TotalCaravanMoney float64 `json:"totalCaravanMoney"`
	DeathsFuel        int     `json:"deathsFuel"`
	DeathsMoney       int     `json:"deathsMoney"`
}

func (s *Server) handleWorld(w http.ResponseWriter, r *http.Request) {
	wj := WorldJSON{
		Day:         s.World.Day,
		Stars:       len(s.World.Stars),
		Planets:     len(s.World.Planets),
		Caravans:    len(s.World.Caravans),
		Pirates:     len(s.World.Pirates),
		Liners:      len(s.World.Liners),
		Diplomats:   len(s.World.Diplomats),
		DeathsFuel:  s.World.DeathsFuel,
		DeathsMoney: s.World.DeathsMoney,
	}
	for _, p := range s.World.Planets {
		wj.TotalPlanetMoney += p.Money
	}
	for _, c := range s.World.Caravans {
		wj.TotalCaravanMoney += c.Money
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(wj)
}

func (s *Server) handleHabitable(w http.ResponseWriter, r *http.Request) {
	ids := []int{}
	for _, p := range s.World.Planets {
		if p.Habitable {
			ids = append(ids, p.ID)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ids)
}

func (s *Server) handleNews(w http.ResponseWriter, r *http.Request) {
	since := 0
	if v := r.URL.Query().Get("since"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			since = n
		}
	}
	out := []sim.Event{}
	for _, e := range s.World.News {
		if e.Day >= since {
			out = append(out, e)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleRelations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.World.Rel)
}