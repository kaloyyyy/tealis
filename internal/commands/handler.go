// in your commands package (e.g., awesomeProject/internal/commands/handler.go)

package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"tealis/internal/storage"
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

	case "EXISTS":
		if len(parts) < 2 {
			return "-ERR EXISTS requires a key"
		}
		key := parts[1]
		if store.Exists(key) {
			return ":1"
		}
		return ":0"

	case "QUIT":
		return "+OK"

	case "APPEND":
		if len(parts) < 3 {
			return "-ERR APPEND requires key and value"
		}
		key, value := parts[1], parts[2]
		newLength := store.Append(key, value)
		return ":" + strconv.Itoa(newLength)

	case "STRLEN":
		if len(parts) < 2 {
			return "-ERR STRLEN requires a key"
		}
		key := parts[1]
		length := store.StrLen(key)
		return ":" + strconv.Itoa(length)

	case "INCR":
		if len(parts) < 2 {
			return "-ERR INCR requires a key"
		}
		key := parts[1]
		newValue, err := store.IncrBy(key, 1)
		if err != nil {
			return "-ERR " + err.Error()
		}
		return ":" + strconv.Itoa(newValue)

	case "DECR":
		if len(parts) < 2 {
			return "-ERR DECR requires a key"
		}
		key := parts[1]
		newValue, err := store.IncrBy(key, -1)
		if err != nil {
			return "-ERR " + err.Error()
		}
		return ":" + strconv.Itoa(newValue)

	case "INCRBY":
		if len(parts) < 3 {
			return "-ERR INCRBY requires a key and increment value"
		}
		key, incrStr := parts[1], parts[2]
		incr, err := strconv.Atoi(incrStr)
		if err != nil {
			return "-ERR Increment must be an integer"
		}
		newValue, err := store.IncrBy(key, incr)
		if err != nil {
			return "-ERR " + err.Error()
		}
		return ":" + strconv.Itoa(newValue)

	case "DECRBY":
		if len(parts) < 3 {
			return "-ERR DECRBY requires a key and decrement value"
		}
		key, decrStr := parts[1], parts[2]
		decr, err := strconv.Atoi(decrStr)
		if err != nil {
			return "-ERR Decrement must be an integer"
		}
		newValue, err := store.IncrBy(key, -decr)
		if err != nil {
			return "-ERR " + err.Error()
		}
		return ":" + strconv.Itoa(newValue)

	case "GETRANGE":
		if len(parts) < 4 {
			return "-ERR GETRANGE requires key, start, and end"
		}
		key, startStr, endStr := parts[1], parts[2], parts[3]
		start, err1 := strconv.Atoi(startStr)
		end, err2 := strconv.Atoi(endStr)
		if err1 != nil || err2 != nil {
			return "-ERR Start and end must be integers"
		}
		result := store.GetRange(key, start, end)
		return "$" + strconv.Itoa(len(result)) + "\r\n" + result

	case "SETRANGE":
		if len(parts) < 4 {
			return "-ERR SETRANGE requires key, offset, and value"
		}
		key, offsetStr, value := parts[1], parts[2], parts[3]
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return "-ERR Offset must be an integer"
		}
		newLength := store.SetRange(key, offset, value)
		return ":" + strconv.Itoa(newLength)

	case "KEYS":
		if len(parts) < 2 {
			return "-ERR KEYS requires a pattern"
		}
		pattern := parts[1]
		keys := store.Keys(pattern)
		if len(keys) == 0 {
			return "(empty list or set)"
		}
		return formatArrayResponse(keys)

		// JSON commands
	case "JSON.SET":
		if len(parts) < 4 {
			return "-ERR JSON.SET requires key, path, and value"
		}
		key, path, value := parts[1], parts[2], parts[3]
		fmt.Printf("")
		fmt.Printf("%s handler set %s %s %s\n", parts, key, path, value)
		err := store.JSONSet(key, path, value)
		if err != nil {
			return "-ERR " + err.Error()
		}
		return "+OK"

	case "JSON.GET":
		if len(parts) < 3 {
			return "-ERR JSON.GET requires key and path"
		}
		key, path := parts[1], parts[2]

		// Retrieve the value from the store
		value, err := store.JSONGet(key, path)
		if err != nil {
			return "-ERR " + err.Error()
		}

		// Serialize the value to a JSON string
		jsonValue, err := json.Marshal(value)
		if err != nil {
			return "-ERR Failed to serialize value to JSON: " + err.Error()
		}

		// Return the serialized value with the correct RESP formatting
		return "$" + strconv.Itoa(len(jsonValue)) + "\r\n" + string(jsonValue)

	case "JSON.DEL":
		if len(parts) < 3 {
			return "-ERR JSON.DEL requires key and path"
		}
		key, path := parts[1], parts[2]

		var err = store.JSONDel(key, path)
		if err != nil {
			return "-ERR " + err.Error()
		}
		return ":1"

	case "JSON.ARRAPPEND":
		if len(parts) < 4 {
			return "-ERR JSON.ARRAPPEND requires key, path, and value(s)"
		}
		key, path, stringVal := parts[1], parts[2], parts[3]
		stringArr := strings.Split(stringVal, ",")
		// Step 2: Convert the slice of strings to a slice of interface{}
		stringInterface := make([]interface{}, len(stringArr))
		for i, v := range stringArr {
			stringInterface[i] = v
		}
		err := store.JSONArrAppend(key, path, stringInterface)
		if err != nil {
			return "-ERR " + err.Error()
		}
		return ":1"
	case "LPUSH":
		if len(parts) < 3 {
			return "-ERR LPUSH requires a key and one or more elements"
		}
		key := parts[1]
		elements := parts[2:]
		newLength := store.LPUSH(key, elements...)
		return fmt.Sprintf(":%d", newLength)

	case "RPUSH":
		if len(parts) < 3 {
			return "-ERR RPUSH requires a key and one or more elements"
		}
		key := parts[1]
		elements := parts[2:]
		newLength := store.RPUSH(key, elements...)
		return fmt.Sprintf(":%d", newLength)

	case "LPOP":
		if len(parts) < 2 {
			return "-ERR LPOP requires a key"
		}
		key := parts[1]
		element, ok := store.LPOP(key)
		if !ok {
			return "$-1"
		}
		return "$" + strconv.Itoa(len(element)) + "\r\n" + element

	case "RPOP":
		if len(parts) < 2 {
			return "-ERR RPOP requires a key"
		}
		key := parts[1]
		element, ok := store.RPOP(key)
		if !ok {
			return "$-1"
		}
		return "$" + strconv.Itoa(len(element)) + "\r\n" + element

	case "LLEN":
		if len(parts) < 2 {
			return "-ERR LLEN requires a key"
		}
		key := parts[1]
		length := store.LLEN(key)
		return fmt.Sprintf(":%d", length)

	case "LRANGE":
		if len(parts) < 4 {
			return "-ERR LRANGE requires a key, start, and end"
		}
		key := parts[1]
		start, err1 := strconv.Atoi(parts[2])
		end, err2 := strconv.Atoi(parts[3])
		if err1 != nil || err2 != nil {
			return "-ERR Start and end must be integers"
		}
		elements := store.LRANGE(key, start, end)
		return formatArrayResponse(elements)
	case "SADD":
		if len(parts) < 3 {
			return "-ERR SADD requires a key and one or more members"
		}
		key := parts[1]
		members := parts[2:]
		addedCount := store.SADD(key, members...)
		return fmt.Sprintf(":%d", addedCount)

	case "SMEMBERS":
		if len(parts) < 2 {
			return "-ERR SMEMBERS requires a key"
		}
		key := parts[1]
		members := store.SMEMBERS(key)
		if len(members) == 0 {
			return "(empty list or set)"
		}
		return formatArrayResponse(members)
	case "SREM":
		if len(parts) < 3 {
			return "-ERR SREM requires a key and one or more members"
		}
		key := parts[1]
		members := parts[2:]
		removedCount := store.SREM(key, members...)
		return fmt.Sprintf(":%d", removedCount)
	case "SISMEMBER":
		if len(parts) < 3 {
			return "-ERR SISMEMBER requires a key and a member"
		}
		key := parts[1]
		member := parts[2]
		exists := store.SISMEMBER(key, member)
		if exists {
			return ":1"
		}
		return ":0"

	default:
		return "-ERR Unknown command"
	}
}

// Helper to format an array response for the KEYS command
func formatArrayResponse(items []string) string {
	var response strings.Builder
	response.WriteString("*" + strconv.Itoa(len(items)) + "\r\n")
	for _, item := range items {
		response.WriteString("$" + strconv.Itoa(len(item)) + "\r\n")
		response.WriteString(item + "\r\n")
	}
	return response.String()
}
