package storage

import "fmt"

// LPush adds one or more elements to the beginning of the list stored at key.
func (r *RedisClone) LPush(key string, values ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	list, ok := r.Store[key].([]string) // Check if the key is already a list.
	if !ok {
		list = []string{} // Initialize if not present.
	}

	// Add elements to the beginning of the list.
	r.Store[key] = append(values, list...)
}

// RPush adds one or more elements to the end of the list stored at key.
func (r *RedisClone) RPush(key string, values ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	list, ok := r.Store[key].([]string)
	if !ok {
		list = []string{}
	}

	// Add elements to the end of the list.
	r.Store[key] = append(list, values...)
}

// LPop removes and returns the first element of the list stored at key.
func (r *RedisClone) LPop(key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	list, ok := r.Store[key].([]string)
	if !ok || len(list) == 0 {
		return "", fmt.Errorf("key does not exist or is not a list")
	}

	// Remove and return the first element.
	first := list[0]
	r.Store[key] = list[1:]
	return first, nil
}

// RPop removes and returns the last element of the list stored at key.
func (r *RedisClone) RPop(key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	list, ok := r.Store[key].([]string)
	if !ok || len(list) == 0 {
		return "", fmt.Errorf("key does not exist or is not a list")
	}

	// Remove and return the last element.
	last := list[len(list)-1]
	r.Store[key] = list[:len(list)-1]
	return last, nil
}
