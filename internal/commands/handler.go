// in your commands package (e.g., awesomeProject/internal/commands/handler.go)

package commands

import (
	"awesomeProject/internal/storage"
	"log"
	"strconv"
	"strings"
	"time"
)

func ProcessCommand(parts []string, store *storage.RedisClone) string {
	if len(parts) == 0 {
		return "-ERR Empty command"
	}

	command := strings.ToUpper(parts[0])
	switch command {
	case "SET":
		if len(parts) < 3 {
			return "-ERR SET requires key and value"
		}
		key, value := parts[1], parts[2]
		var ttl time.Duration
		if len(parts) == 5 && strings.ToUpper(parts[3]) == "EX" {
			ttlSecs, err := strconv.Atoi(parts[4])
			if err != nil {
				return "-ERR Invalid TTL"
			}
			ttl = time.Duration(ttlSecs) * time.Second
			log.Printf("SET expiry: %d", ttlSecs)
		}
		store.Set(key, value, ttl)
		return "+OK"
	case "GET":
		if len(parts) < 2 {
			return "-ERR GET requires a key"
		}
		key := parts[1]
		value, exists := store.Get(key)
		if !exists {
			return "$-1"
		}
		return "$" + strconv.Itoa(len(value)) + "\r\n" + value
	case "DEL":
		if len(parts) < 2 {
			return "-ERR DEL requires a key"
		}
		key := parts[1]
		if store.Del(key) {
			return ":1"
		}
		return ":0"
	default:
		return "-ERR Unknown command"
	}
}
