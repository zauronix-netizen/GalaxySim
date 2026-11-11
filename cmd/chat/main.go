package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ChatMessage struct {
	ID   int64  `json:"id"`
	Day  int    `json:"day"`
	Nick string `json:"nick"`
	Text string `json:"text"`
	Ts   int64  `json:"ts"`
}

const (
	baseURL = "http://localhost:8080"
	room    = "world"
	nick    = "admin"
)

var httpClient = &http.Client{Timeout: 5 * time.Second}

func fetchSince(since int64) ([]ChatMessage, error) {
	url := fmt.Sprintf("%s/chat/messages?room=%s&since=%d", baseURL, room, since)
	r, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", r.StatusCode)
	}
	var out []ChatMessage
	if err := json.NewDecoder(r.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func postMessage(text string) error {
	body, _ := json.Marshal(map[string]string{"room": room, "nick": nick, "text": text})
	r, err := httpClient.Post(baseURL+"/chat/messages", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(r.Body)
		return fmt.Errorf("status %d: %s", r.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func clearLine() {
	// Стираем строку без ANSI (работает в старом cmd.exe).
	fmt.Print("\r" + strings.Repeat(" ", 100) + "\r")
}

func main() {
	fmt.Printf("chat client | %s | room=%s | nick=%s\n", baseURL, room, nick)
	fmt.Println("вводи сообщение + Enter. /quit — выход.")
	fmt.Print("> ")

	lastID := int64(0)
	msgCh := make(chan ChatMessage, 64)

	// polling — раз в 2 сек, ошибки не спамим (не чаще раза в 30 сек)
	go func() {
		var lastErr time.Time
		for {
			msgs, err := fetchSince(lastID)
			if err == nil {
				for _, m := range msgs {
					if m.ID > lastID {
						lastID = m.ID
						msgCh <- m
					}
				}
			} else if time.Since(lastErr) > 30*time.Second {
				msgCh <- ChatMessage{Nick: "[err]", Text: err.Error()}
				lastErr = time.Now()
			}
			time.Sleep(2 * time.Second)
		}
	}()

	// чтение stdin — построчно
	lineCh := make(chan string)
	go func() {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			lineCh <- sc.Text()
		}
		close(lineCh)
	}()

	for {
		select {
		case m, ok := <-msgCh:
			if !ok {
				return
			}
			clearLine()
			fmt.Printf("%s: %s\n> ", m.Nick, m.Text)

		case line, ok := <-lineCh:
			if !ok {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				fmt.Print("> ")
				continue
			}
			if line == "/quit" || line == "/q" {
				return
			}
			if err := postMessage(line); err != nil {
				clearLine()
				fmt.Printf("[err] %v\n> ", err)
				continue
			}
			fmt.Print("> ")
		}
	}
}