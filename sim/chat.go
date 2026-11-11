package sim

import (
	"sync"
	"time"
)

// ChatMessage — одно сообщение в комнате чата.
type ChatMessage struct {
	ID   int64  `json:"id"`
	Day  int    `json:"day"`
	Nick string `json:"nick"`
	Text string `json:"text"`
	Ts   int64  `json:"ts"` // unix ms, реальное время
}

// ChatStore — чат в памяти. Ring buffer на комнату.
type ChatStore struct {
	mu       sync.Mutex
	next     int64
	rooms    map[string][]ChatMessage
	capacity int
}

func NewChatStore(capacity int) *ChatStore {
	if capacity < 1 {
		capacity = 100
	}
	return &ChatStore{
		rooms:    make(map[string][]ChatMessage),
		capacity: capacity,
	}
}

// Post добавляет сообщение в комнату. Возвращает сохранённое (с ID).
func (c *ChatStore) Post(room, nick, text string, day int) ChatMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.next++
	msg := ChatMessage{
		ID:   c.next,
		Day:  day,
		Nick: nick,
		Text: text,
		Ts:   time.Now().UnixMilli(),
	}

	buf := c.rooms[room]
	buf = append(buf, msg)
	if len(buf) > c.capacity {
		// отрезаем старое; копируем, чтобы не держать большой массив
		cut := len(buf) - c.capacity
		buf = append([]ChatMessage(nil), buf[cut:]...)
	}
	c.rooms[room] = buf
	return msg
}

// Since возвращает сообщения комнаты с ID > since (до capacity штук).
func (c *ChatStore) Since(room string, since int64) []ChatMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	buf := c.rooms[room]
	if len(buf) == 0 {
		return []ChatMessage{}
	}
	out := make([]ChatMessage, 0, len(buf))
	for _, m := range buf {
		if m.ID > since {
			out = append(out, m)
		}
	}
	return out
}