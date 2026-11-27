# GalaxySim

**MMO-экономика на движке Space Rangers 2: Reboot / HD (Космические рейнджеры 2: Революция).**

> Симулятор, не игра. Канон — фундамент, но **свой мир**.

---

## Что работает

- **Полная генерация галактики** — 600 звёзд, ~4000 планет, было 9×9 = 81 секторов - после тестов 9х8, созвездия (MST), соседи.
- **38 товаров** (Elite-порядок), 9 экономик, 10 политик, 5 рас — с каноническими коэффициентами производства/потребления.
- **Экономика планет** — `TargetStock × TL`, динамические цены (`UpdateMarketState`), налог с населения, спецтовары по типу планеты.
- **Паттерн-караваны (v18)** — планеты считают `ProducedMask`/`ConsumedMask` (`uint64`, 38 бит) из `prod/cons` раз в 144 тика. Караваны покупают **только локально** по паттерну (`ProducedMask` + склад > 0), продают по паттерну дефицита (`ConsumedMask`) с очерёдностью: своя система → своё созвездие → вся карта. Ясновидения нет — не видят чужие склады/цены.
- **Биржевой стакан (v19)** — `SystemMarketSnapshot`: агрегированный спрос/предложение по системам, пересчёт раз в день. Караваны в `caravanPlanRoute`/`caravanFindDeficitNearby` сначала смотрят в снапшоты — сужают поиск до реальных пар.
- **История каравана (v20)** — кольцевой буфер 1500 записей (`buyPrice`/`sellPrice`, `fineTrade`, `reason`, `day`). Причины: `""`, `"pirate"`, `"price"`, `"no_money"`, `"no_goods"`, `"no_cargo"`. Экспорт: `GET /caravan/<id>` + CLI `caravan-history.exe`.
- **Банкротство вместо смерти** — если денег нет на закупку: новые 3000..7000, `TL = planet.TL - 1`, сброс груза, `PhaseFindTrip`.
- **Пираты** — привязаны к логовам (`oiPirate`), грабят/убивают караваны в своей системе. Деньги/груз логову → апгрейд пиратов (`Wins`, `TechLevel`).
- **Трейдеры** — курьеры по новостям (телепорт к избытку, поиск дефицита). Есть история сделок (в коде, эндпоинт отключён).
- **Станции игрока** — 72 (по одной на сектор), 8 категорий товаров, баланс/маржи, API для торговли.
- **MMO-скелет** — `ShipHub` (потокобезопасный), `Ship` с векторной позицией/скоростью, команды `MoveOrbit`/`MoveGoto`/`MoveEdge`/`MoveDock`/`MoveUndock`, ETA = `ceil(dist/ShipSpeed)`. `ShipSpeed = 0.5 / TicksPerDay`.
- **Чат + карта системы** — `chat.html` (flex: чат слева, карта 600×600 справа): звезда, орбиты, планеты, корабль, Безье-траектория, клик-инфо, команды `/spawn`, `/goto`, `/orbit`, `/undock`, `/despawn`.
- **Админ** — `/admin?ms=` (localhost only), override тика `0=auto, 1..100000` мс.
- **Три режима тика**: `Fast` (1 тик = 1 день, 100 мс, до GTL 4), `Slow` (календарь 2048 тиков/день, 10 мс, после GTL 4), `SuperFast` (1 тик = 1 час, 10 мс, без кораблей).

---

## Что НЕ работает / в разработке

| Проблема | Статус |
|----------|--------|
| `MoveOrbit` — `Pos` не ставится (заглушка) | Открыто |
| `Docked=true` при `spawn`→`goto` (баг) | Открыто |
| Кнопки `dock`/`orbit` в UI | Отсутствуют |
| Чтение `ShipHub` в HTTP без `Mu` (гонки) | Известно |
| Падение на `buildRoute=nil` | Известно |
| Деньги не сохраняются (×2.8 за 100 лет) | Отложено |
| Население не меняется (`Pop` статично) | Отложено |
| Производство почти ничего (раз в 144 тика) | Осознанно |
| Отношения не возвращаются (`/20` целочисленно) | Отложено |
| Лайнеры/дипломаты отключены (`target=0`) | Мёртвый код |

---

## Быстрый старт

```bash
# Требования: Go 1.22+ (1.27.4 рекомендуется)
git clone https://github.com/zauronix-netizen/GalaxySim
cd GalaxySim

# Сборка
go build .

# Запуск (рандомный сид)
./galaxysim.exe

# Или с фиксированным сидом для воспроизводимости
./galaxysim.exe -seed=12345
```

**Флаги:**
- `-seed=N` — сид генерации мира (`0` = `time.Now().UnixNano()`)
- `-viewer-star=N` — звезда для Shared Memory viewer (`-1` = случайная)

После запуска:
- **HTTP API:** `http://localhost:8080`
- **Чат/Карта:** `http://localhost:8080/chat`
- **Админ (override тика):** `http://localhost:8080/admin?ms=10` (только localhost)

---

## API (основные эндпоинты)

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/world` | Сводка: день, GTL, деньги по группам, счетчики агентов |
| GET | `/map` | Карта секторов/звёзд (для чата) |
| GET | `/sector/{id}` | Рыночные средние по сектору |
| GET | `/planet/{id}` | Полная инфа планеты: рынок, население, ТЛ, раса/политика |
| GET | `/system/{id}` | Планеты в системе |
| GET | `/habitable` | Список обитаемых планет |
| GET | `/lair/{id}` | Логово пиратов (груз, деньги, ТЛ) |
| GET | `/caravan/{id}` | **Новое:** данные каравана + история сделок (`buyPrice`/`sellPrice`) |
| GET | `/news` | Игровые новости (дефицит/избыток) |
| GET | `/relations` | Матрица отношений рас |
| GET | `/mmo/state?shipId=N` | Состояние корабля (поз, скорость, ETA, док) |
| GET | `/mmo/ships?system=N` | Все корабли в системе |
| POST | `/mmo/command` | Команда кораблю (`MoveOrbit`, `MoveGoto`, `MoveEdge`, `MoveDock`, `MoveUndock`) |
| POST | `/mmo/spawn` | Спавн корабля на станции/планете |
| POST | `/mmo/despawn` | Удаление корабля |
| POST | `/station/trade` | Торговля на станции игрока (`sector`, `good`, `qty`, `op:buy/sell`) |
| GET/POST | `/chat/messages` | Чат (комнаты, ник, история) |

---

## CLI утилиты (в `cmd/`)

```bash
# История каравана
go build ./cmd/caravan-history
./caravan-history.exe -id 123

# Чат-клиент
go build ./cmd/chat
./chat.exe
```

---

## Архитектура

```
galaxysim/
├── main.go                 # Entry point, HTTP server, тикер
├── go.mod
├── api/
│   └── http.go             # Все HTTP хендлеры (JSON API)
├── sim/
│   ├── world.go            # World, Planet, Star, MarketEntry, SystemMarketSnapshot
│   ├── agents.go           # Caravan, Trader, Pirate, Liner, Diplomat, HistoryEntry
│   ├── caravan.go          # Логика караванов: спаун, маршруты, загрузка/разгрузка, история
│   ├── trader.go           # Трейдеры: фазы, новости, пиратский цикл, история
│   ├── pirate.go           # Пираты: фазы, атака, апгрейд логова
│   ├── tick.go             # SimTickFast/Slow/SuperFast, фазы: Movement, Spawn, Economy, Stations, Traders
│   ├── market.go           # MarketEntry, UpdateMarketState, цены, легальность
│   ├── trade.go            # Buy/Sell, CanTrade, ShopGoodsPurchasePrice/SellPrice
│   ├── production.go       # applyProduction, Prod/Cons, маски, налог, спецтовары
│   ├── flight.go           # FlightPath, безье, общие функции движения
│   ├── route.go            # buildRoute (BFS по DistMatrix с jumpRange)
│   ├── mmo.go              # Vec2, Ship, MoveKind, MoveOrder, ShipCommand
│   ├── mmo_hub.go          # ShipHub (Mu, Spawn, ApplyCommand, Tick)
│   ├── dashboard.go        # Консольный дашборд (ansi)
│   ├── consts.go           # GoodsMarket, эконом/полит/расовые таблицы, константы
│   ├── galtech.go          # computeGalaxyTechLevel (GTL)
│   ├── econ_summary.go     # Статистика, dayToDate
│   └── ...                 # liner, diplomat, sectors, relations, tickEvents
├── cmd/
│   ├── caravan-history/    # CLI чтение истории каравана
│   ├── chat/               # CLI чат-клиент
│   └── ...                 # viewer, lairs
├── chat.html               # Чат + карта (go:embed)
├── index.html              # Shell (go:embed)
└── README.md
```

---

## Календарь

- **36 дней в месяце, 16 месяцев в году** (hex `0x00`–`0x0F`), год с 3300.
- `daysInYear = 576`. `TurnsPerYear = 576`. `GalaxyWarmupTurns = 576`.
- Формат: `dd/0xMM/yyyy`. `dayToDate` в `econ_summary.go` (и JS).
- `MinutesPerHour = 64`, `HoursPerDay = 32`, `TicksPerDay = 2048`.
- **Warmup:** 1 тик = 1 день. **После:** `Mins++` → `Hour++` → `Day++`. **2048 тиков/день.**

---

## Деньги и печать

- **Караваны** — встроенный спред: покупка `cost * 0.975` (97.5%), продажа `rev * 1.025` (102.5%) → **~5% на оборот**. Без этого караваны гниют массово (`DeathsMoney`).
- **Банкротство** — вместо смерти: новые деньги, `TL = planet.TL - 1`, сброс груза/фазы.
- **Налог в `applyProduction`** — мёртвый путь (`if p.Money < p.MoneyRef { tax := Pop/100 }`), не работает.
- **Полный аудит `Money +=` не делался** — «не проверено».

---

## RAM-диск (R:) — обязательно для комфорта на HDD

| Что | Значение |
|-----|----------|
| Инструмент | ImDisk 2.1.1 (ImDisk Toolkit 20250206) |
| Размер | 4 ГБ NTFS |
| Переживает ребут | Да (образ сохраняется/восстанавливается) |
| `GOCACHE` | `R:\go-build` |
| `GOTMPDIR` | `R:\gotmp` |
| Холодная сборка | 109 с (2 ядра, HDD) |
| Кэш после сборки | 94 МБ, 1436 файлов |

> `RamDyn.exe` (динамический режим toolkit) **не работает** без GUI-мастера и службы `ImDiskTk-svc.exe`. Используем статический диск.

---

## Технологии

- **Go 1.22+** (`net/http`, `math/rand`, `encoding/json`, `sync`, `time`)
- **Ванильный JS** (`chat.html`, `index.html` — `//go:embed`)
- **ANSI-дашборд** в консоли (`dashboard.go`)
- **Shared Memory** (опционально, для `viewer.exe`)

---

## Лицензия

MIT. Автор: `ZauronixRus`.

> Проект неофициальный, не связан с 1C / Elemental Games / SNEG. Используется декомпиляция `Space Rangers HD: A War Apart` (v2.1.2500) как референс для игровых формул/данных.

---

## Планы (high-level)

1. **Играбельная проверка** — паттерн-караваны + биржевой стакан в игре (`took`, простой, экономика).
2. **Пиратский риск в маршрутах** — штраф/бан за пиратские звёзды в `buildRoute`.
3. **MMO-движок** — `MoveOrbit` fix, `Docked` при командах, гравитация/ETA, NPC/бой.
4. **Экономика** — замкнутые деньги, динамическое население, сохранение мира.
5. **Клиент** — 3D (Raylib/Godot/Rust), кроссплатформенный (ПК + Android).
6. **Релиз** — полная переработка кода (SOLID, модули), чистый `v1.0.0`.

---

## Вклад

PR приветствуются, но:
- **Сначала обсуждение** (Issue) — что меняется, цена, риск, откат.
- **Один шаг** — правка → проверка → следующий.
- **Никаких больших рефакторингов без согласования** — проект в режиме «слесарная починка».

---

## Контакты

- GitHub: `zauronix-netizen/GalaxySim`
- Issues / Discussions — для вопросов и багов.