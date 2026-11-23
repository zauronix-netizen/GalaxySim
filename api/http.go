package api

import (
	_ "embed"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"io"
	"fmt"
	"regexp"
	"strings"
	"galaxysim/sim"
)

//go:embed index.html
var indexHTML []byte

//go:embed chat.html
var chatHTML []byte

type Server struct {
	World *sim.World
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/sector/", s.handleSector)
	mux.HandleFunc("/planet/", s.handlePlanet)
	mux.HandleFunc("/player/", s.handlePlayer)
	mux.HandleFunc("/station/", s.handleStation)       // уже есть (GET)
	mux.HandleFunc("/station/trade", s.handleStationTrade) // POST
	mux.HandleFunc("/map", s.handleMap)
	mux.HandleFunc("/world", s.handleWorld)
	mux.HandleFunc("/habitable", s.handleHabitable)
	mux.HandleFunc("/news", s.handleNews)
	mux.HandleFunc("/relations", s.handleRelations)
	mux.HandleFunc("/system/", s.handleSystem)
	mux.HandleFunc("/lair/", s.handleLair)
	mux.HandleFunc("/chat", s.handleChatPage)
	mux.HandleFunc("/chat/messages", s.handleChatMessages)
	mux.HandleFunc("/admin", s.handleAdmin)
	mux.HandleFunc("/mmo/state", s.handleMMOState)
	mux.HandleFunc("/mmo/command", s.handleMMOCommand)
	mux.HandleFunc("/mmo/spawn", s.handleMMOSpawn)
	mux.HandleFunc("/mmo/despawn", s.handleMMODespawn)
	mux.HandleFunc("/mmo/ships", s.handleMMOShips)
	mux.HandleFunc("/mmo/systemobj", s.handleSystemObjects)
	mux.HandleFunc("/", s.handleIndex)
}

type MMOShipBriefJSON struct {
	ShipID      int      `json:"shipId"`
	PlayerID    int      `json:"playerId"`
	Pos         sim.Vec2 `json:"pos"`
	Docked      bool     `json:"docked"`
	DockID      int      `json:"dockId"`
	OrbitPlanet int      `json:"orbitPlanet"`
	OrbitRadius float64  `json:"orbitRadius"`
	ETA         int      `json:"eta"`
	Kind        int      `json:"kind"`
}

type MMOStateJSON struct {
	ShipID      int      `json:"shipId"`
	PlayerID    int      `json:"playerId"`
	SystemID    int      `json:"systemId"`
	Pos         sim.Vec2 `json:"pos"`
	Vel         sim.Vec2 `json:"vel"`
	Docked      bool     `json:"docked"`
	DockID      int      `json:"dockId"`
	OrbitPlanet int      `json:"orbitPlanet"`
	OrbitRadius float64  `json:"orbitRadius"`
	ETA         int      `json:"eta"`
	Kind        int      `json:"kind"`
	Target      *sim.Vec2 `json:"target,omitempty"`
	Start       *sim.Vec2 `json:"start,omitempty"`
}

type MMOCommandJSON struct {
	PlayerID int      `json:"playerId"`
	ShipID   int      `json:"shipId"`
	Kind     int      `json:"kind"`
	Planet   int      `json:"planet"`
	Radius   float64  `json:"radius"`
	Target   sim.Vec2 `json:"target"`
}

type MMOSpawnJSON struct {
	PlayerID int `json:"playerId"`
	SystemID int `json:"systemId"`
	DockID   int `json:"dockId"`
}

type SectorMarketJSON struct {
	Good  int     `json:"good"`
	Buy   float64 `json:"buy"`
	Sell  float64 `json:"sell"`
	Count int     `json:"count"`
	Stock int     `json:"stock"`
}

type StationGoodJSON struct {
	Good    int     `json:"good"`
	Buy     float64 `json:"buy"`
	Sell    float64 `json:"sell"`
	Count   int     `json:"count"`
	Stock   int     `json:"stock"`
	Balance int     `json:"balance"`
}

type stationTradeReq struct {
	Sector int    `json:"sector"`
	Good   int    `json:"good"`
	Qty    int    `json:"qty"`
	Op     string `json:"op"` // "buy" | "sell"
}

type SectorJSON struct {
	Sector      int                `json:"sector"`
	Stars       int                `json:"stars"`
	Planets     int                `json:"planets"`
	Uninhabited int                `json:"uninhabited"`
	Lairs       int                `json:"lairs"`
	Market      []SectorMarketJSON `json:"market"`
}

type LairJSON struct {
	ID        int     `json:"id"`
	StarID    int     `json:"starId"`
	Orbit     float64 `json:"orbit"`
	Race      int     `json:"race"`
	Money     float64 `json:"money"`
	TechLevel int     `json:"techLevel"`
	Cargo     []int   `json:"cargo"` // порядок GoodsTextOrder
}

func (s *Server) handleMMOShips(w http.ResponseWriter, r *http.Request) {
	sysStr := r.URL.Query().Get("system")
	sysID, err := strconv.Atoi(sysStr)
	if err != nil {
		http.Error(w, "bad system id", http.StatusBadRequest)
		return
	}
	s.World.Ships.Mu.Lock()
	out := make([]MMOShipBriefJSON, 0, len(s.World.Ships.Ships))
	for _, sh := range s.World.Ships.Ships {
		if sh.SystemID != sysID {
			continue
		}
		b := MMOShipBriefJSON{
			ShipID: sh.ID, PlayerID: sh.PlayerID, Pos: sh.Pos,
			Docked: sh.Docked, DockID: sh.DockID,
			OrbitPlanet: sh.OrbitPlanet, OrbitRadius: sh.OrbitRadius,
		}
		if sh.Target != nil {
			b.ETA = sh.Target.ETA
			b.Kind = int(sh.Target.Kind) + 1
		}
		out = append(out, b)
	}
	s.World.Ships.Mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// handleSystemObjects — корабли в системе для карты чата: позиция + цель.
// Планеты и звезду карта знает из /system, здесь только движущиеся корабли.
func (s *Server) handleSystemObjects(w http.ResponseWriter, r *http.Request) {
	sysID, err := strconv.Atoi(r.URL.Query().Get("system"))
	if err != nil {
		http.Error(w, "bad system id", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"day":   s.World.Day,
		"star":  sysID,
		"ships": s.World.MapShips(sysID),
	})
}

func (s *Server) handleMMOState(w http.ResponseWriter, r *http.Request) {
	shipIDStr := r.URL.Query().Get("ship")
	shipID, err := strconv.Atoi(shipIDStr)
	if err != nil {
		http.Error(w, "bad ship id", http.StatusBadRequest)
		return
	}
	sh, ok := s.World.Ships.Ships[shipID]
	if !ok {
		http.Error(w, "ship not found", http.StatusNotFound)
		return
	}
	out := MMOStateJSON{
		ShipID: sh.ID, PlayerID: sh.PlayerID, SystemID: sh.SystemID,
		Pos: sh.Pos, Vel: sh.Vel, Docked: sh.Docked, DockID: sh.DockID,
		OrbitPlanet: sh.OrbitPlanet, OrbitRadius: sh.OrbitRadius,
	}
	if sh.Target != nil {
		out.ETA = sh.Target.ETA
		out.Kind = int(sh.Target.Kind) + 1
		t := sh.Target.Target
		out.Target = &t
		st := sh.Target.Start
		out.Start = &st
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleMMOCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	var req MMOCommandJSON
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	cmd := sim.ShipCommand{
		PlayerID: req.PlayerID, ShipID: req.ShipID,
		Kind: sim.MoveKind(req.Kind), Planet: req.Planet,
		Radius: req.Radius, Target: req.Target,
	}
	sh, ok := s.World.Ships.ApplyCommand(s.World, cmd)
	if !ok {
		http.Error(w, "ship not found", http.StatusNotFound)
		return
	}
	eta := 0
	if sh.Target != nil {
		eta = sh.Target.ETA
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": true, "shipId": sh.ID, "eta": eta,
	})
}

func (s *Server) handleMMOSpawn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	var req MMOSpawnJSON
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	sh := s.World.Ships.Spawn(req.PlayerID, req.SystemID, req.DockID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": true, "shipId": sh.ID, "systemId": sh.SystemID, "dockId": sh.DockID,
	})
}

func (s *Server) handlePlayer(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/player/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= sim.SectorCount {
		http.Error(w, "bad sector id", http.StatusBadRequest)
		return
	}
	market := s.World.PlayerMarket(id)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(market)
}

func (s *Server) handleSector(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/sector/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= sim.SectorCount {
		http.Error(w, "bad sector id", http.StatusBadRequest)
		return
	}

	out := SectorJSON{
		Sector: id,
		Market: make([]SectorMarketJSON, 0, len(sim.GoodsTextOrder)),
	}

	type acc struct {
		sumBuy   float64
		sumSell  float64
		sumCount int
		sumStock int
		n        int
	}
	var mkt [sim.GoodCount]acc

	for _, st := range s.World.Stars {
		if st.Sector != id {
			continue
		}
		out.Stars++
		for _, p := range s.World.PlanetMap[st.ID] {
			out.Planets++
			if !p.Habitable {
				out.Uninhabited++
			}
			if p.IsPirateLair() {
				out.Lairs++
			}
			if !p.Habitable || p.IsPirateLair() {
				continue
			}
			for g := 0; g < int(sim.GoodCount); g++ {
				m := &p.Market[g]
				mkt[g].sumBuy += m.PurchasePrice
				mkt[g].sumSell += m.BaseSalePrice
				mkt[g].sumCount += m.Count
				mkt[g].sumStock += m.TargetStock
				mkt[g].n++
			}
		}
	}

	for _, idx := range sim.GoodsTextOrder {
		a := mkt[idx]
		buy, sell := 0.0, 0.0
		if a.n > 0 {
			buy = a.sumBuy / float64(a.n)
			sell = a.sumSell / float64(a.n)
		}
		out.Market = append(out.Market, SectorMarketJSON{
			Good:  idx,
			Buy:   buy,
			Sell:  sell,
			Count: a.sumCount,
			Stock: a.sumStock,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleLair(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/lair/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= len(s.World.Planets) {
		http.Error(w, "bad planet id", http.StatusBadRequest)
		return
	}
	p := s.World.Planets[id]
	if !p.IsPirateLair() {
		http.Error(w, "not a lair", http.StatusNotFound)
		return
	}
	out := LairJSON{
		ID:        p.ID,
		StarID:    p.StarID,
		Orbit:     p.Orbit,
		Race:      int(p.Race),
		Money:     p.Money,
		TechLevel: p.TechLevel,
		Cargo:     make([]int, 0, len(sim.GoodsTextOrder)),
	}
	for _, idx := range sim.GoodsTextOrder {
		out.Cargo = append(out.Cargo, p.Market[idx].Count)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleMMODespawn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1024))
	var req struct{ ShipID int `json:"shipId"` }
	if err := json.Unmarshal(body, &req); err != nil || req.ShipID <= 0 {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	delete(s.World.Ships.Ships, req.ShipID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "shipId": req.ShipID})
}

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {

	// Только localhost (127.0.0.1 / ::1).
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || (host != "127.0.0.1" && host != "::1") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// ... остальное как есть

	v := r.URL.Query().Get("ms")
	if v == "" {
		http.Error(w, "usage: /admin?ms=100 (0=auto)", http.StatusBadRequest)
		return
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 || n > 100000 {
		http.Error(w, "bad ms (0=auto, 1..100000)", http.StatusBadRequest)
		return
	}
	s.World.TickOverrideMs = n
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"ms":%d}`, n)
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

type SystemPlanetJSON struct {
	ID        int     `json:"id"`
	Orbit     float64 `json:"orbit"`
	Habitable bool    `json:"habitable"`
	Race      int     `json:"race"`
	TechLevel int     `json:"techLevel"`
	Pop       int     `json:"pop"`
	IsPirate  bool    `json:"isPirate"`
}

type SystemJSON struct {
	StarID       int                `json:"starId"`
	Sector       int                `json:"sector"`
	SystemRadius float64            `json:"systemRadius"`
	Planets      []SystemPlanetJSON `json:"planets"`
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

func (s *Server) handleSystem(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/system/"):]
	starID, err := strconv.Atoi(idStr)
	if err != nil || starID < 0 || starID >= len(s.World.Stars) {
		http.Error(w, "bad star id", http.StatusBadRequest)
		return
	}
	st := s.World.Stars[starID]
	out := SystemJSON{
		StarID:       starID,
		Sector:       st.Sector,
		SystemRadius: st.SystemRadius,
		Planets:      make([]SystemPlanetJSON, 0, len(s.World.PlanetMap[starID])),
	}
	for _, p := range s.World.PlanetMap[starID] {
		out.Planets = append(out.Planets, SystemPlanetJSON{
			ID:        p.ID,
			Orbit:     p.Orbit,
			Habitable: p.Habitable,
			Race:      int(p.Race),
			TechLevel: p.TechLevel,
			Pop:       p.Pop,
			IsPirate:  p.IsPirateLair(),
		})
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
	case sim.EcoAgriPoor:
		return "Agr-p"
	case sim.EcoAgriAvg:
		return "Agr-a"
	case sim.EcoAgriRich:
		return "Agr-r"
	case sim.EcoMixedPoor:
		return "Mix-p"
	case sim.EcoMixedAvg:
		return "Mix-a"
	case sim.EcoMixedRich:
		return "Mix-r"
	case sim.EcoIndPoor:
		return "Ind-p"
	case sim.EcoIndAvg:
		return "Ind-a"
	case sim.EcoIndRich:
		return "Ind-r"
	}
	return "?"
}

func economyName(e sim.Economy) string {
	return economyShort(e)
}

func governmentShort(g sim.Government) string {
	switch g {
	case sim.GovAnarchy:
		return "anar"
	case sim.GovFeudal:
		return "feud"
	case sim.GovMultiGovernment:
		return "multi"
	case sim.GovDictatorship:
		return "dict"
	case sim.GovCommunist:
		return "com"
	case sim.GovConfederacy:
		return "conf"
	case sim.GovDemocracy:
		return "dem"
	case sim.GovCorporateState:
		return "corp"
	case sim.GovMonarchy:
		return "mon"
	case sim.GovRepublic:
		return "rep"
	}
	return "?"
}

func governmentName(g sim.Government) string {
	switch g {
	case sim.GovAnarchy:
		return "anarchy"
	case sim.GovFeudal:
		return "feudal"
	case sim.GovMultiGovernment:
		return "multi-government"
	case sim.GovDictatorship:
		return "dictatorship"
	case sim.GovCommunist:
		return "communist"
	case sim.GovConfederacy:
		return "confederacy"
	case sim.GovDemocracy:
		return "democracy"
	case sim.GovCorporateState:
		return "corporate state"
	case sim.GovMonarchy:
		return "monarchy"
	case sim.GovRepublic:
		return "republic"
	}
	return "?"
}

func (s *Server) handleStation(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/station/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= sim.SectorCount {
		http.Error(w, "bad sector id", http.StatusBadRequest)
		return
	}
	if id >= len(s.World.Stations) || s.World.Stations[id] == nil {
		http.Error(w, "station not found", http.StatusNotFound)
		return
	}
	st := s.World.Stations[id]
	pm := s.World.PlayerMarket(id)

	// Сток по категориям игрока: сумма TargetStock по маппингу PlayerGoodMap,
	// по всем обитаемым непиратским планетам сектора.
	var stock [sim.PlayerGoodCount]int
	for _, stp := range s.World.Stars {
		if stp.Sector != id {
			continue
		}
		for _, p := range s.World.PlanetMap[stp.ID] {
			if !p.Habitable || p.IsPirateLair() {
				continue
			}
			for pg := 0; pg < int(sim.PlayerGoodCount); pg++ {
				for _, g := range sim.PlayerGoodMap[pg] {
					stock[pg] += p.Market[g].TargetStock
				}
			}
		}
	}

	st.Mu.Lock()
	out := make([]StationGoodJSON, 0, sim.PlayerGoodCount)
	for pg := 0; pg < int(sim.PlayerGoodCount); pg++ {
		buy, sell := s.World.StationPrice(sim.PlayerGood(pg), pm[pg].Count, stock[pg])
		out = append(out, StationGoodJSON{
			Good:    pg,
			Buy:     buy,
			Sell:    sell,
			Count:   pm[pg].Count,
			Stock:   stock[pg],
			Balance: st.Balance[pg],
		})
	}
	st.Mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleStationTrade(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	var req stationTradeReq
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.Sector < 0 || req.Sector >= sim.SectorCount ||
		req.Good < 0 || req.Good >= int(sim.PlayerGoodCount) ||
		req.Qty < 1 || req.Qty > 10000 {
		http.Error(w, "bad params", http.StatusBadRequest)
		return
	}
	st := s.World.Stations[req.Sector]
	if st == nil {
		http.Error(w, "station not found", http.StatusNotFound)
		return
	}
	st.Mu.Lock()
	switch req.Op {
	case "buy":
		st.Balance[req.Good] -= req.Qty // игрок купил → станция должна
	case "sell":
		st.Balance[req.Good] += req.Qty // игрок продал → планеты должны станции
	default:
		st.Mu.Unlock()
		http.Error(w, "bad op", http.StatusBadRequest)
		return
	}
	bal := st.Balance[req.Good]
	st.Mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"balance": bal})
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
	Hour              int     `json:"hour"`
	Mins              int     `json:"mins"`
	GalTech           int     `json:"galTech"`
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
		Hour:        s.World.Hour,
		Mins:        s.World.Mins,
		GalTech:     s.World.GalaxyTechLevel,
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

func (s *Server) handleChatPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/chat" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(chatHTML)
}

var chatRoomRe = regexp.MustCompile(`^(world|sector:[0-9]+|system:[0-9]+|planet:[0-9]+)$`)

func validChatRoom(room string) bool {
	return chatRoomRe.MatchString(room)
}

type chatPostReq struct {
	Room string `json:"room"`
	Nick string `json:"nick"`
	Text string `json:"text"`
}

func (s *Server) handleChatMessages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		room := r.URL.Query().Get("room")
		if !validChatRoom(room) {
			http.Error(w, "bad room", http.StatusBadRequest)
			return
		}
		var since int64
		if v := r.URL.Query().Get("since"); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				since = n
			}
		}
		out := s.World.Chat.Since(room, since)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)

	case http.MethodPost:
		body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
		var req chatPostReq
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if !validChatRoom(req.Room) {
			http.Error(w, "bad room", http.StatusBadRequest)
			return
		}
		nick := strings.TrimSpace(req.Nick)
		if nick == "" {
			nick = "anon"
		}
		if len(nick) > 20 {
			nick = nick[:20]
		}
		text := strings.TrimSpace(req.Text)
		if text == "" || len(text) > 200 {
			http.Error(w, "bad text", http.StatusBadRequest)
			return
		}
		msg := s.World.Chat.Post(req.Room, nick, text, s.World.Day)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(msg)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

