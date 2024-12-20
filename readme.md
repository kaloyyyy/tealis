# Tealis CLI Commands
`telnet 127.0.0.1 6379` in CMD to connect
`localhost:8000` open in your browser
`localhost:8081` for HTTP API requests
## General Commands
- `MULTI` - Marks the start of a transaction.
- `EXEC` - Executes all commands issued after `MULTI`.
- `DISCARD` - Discards all commands issued after `MULTI`.
- `SET [key] [value]` - Sets a key to hold a string value. Example: `SET mykey "sample value"`.
- `GET [key]` - Gets the value of a key if it exists.
- `DEL [key]` - Deletes a key.
- `EXISTS [key]` - Checks if a key exists (returns 1 if it does, 0 otherwise).
- `EX [key] [time_in_sec]` - Sets a timeout on a key.
- `TTL [key]` - Returns the remaining time-to-live of a key.
- `PERSIST [key]` - Removes the expiration from a key.
- `QUIT` - Disconnects the client from the server.
- `SAVE` - Saves the dataset to disk synchronously.
- `BGSAVE` - Saves the dataset to disk asynchronously in the background.
- `RESTORE` - Restores a key from a dump file.
- `AOF` - Checks if AOF persistence is enabled.
- `APPEND [key] [value]` - Appends a value to an existing string.
- `STRLEN [key]` - Gets the length of the string value stored in a key.
- `INCR [key]` - Increments the integer value of a key by 1.
- `DECR [key]` - Decrements the integer value of a key by 1.
- `INCRBY [key] [value]` - Increments the integer value of a key by the specified value.
- `DECRBY [key] [value]` - Decrements the integer value of a key by the specified value.
- `GETRANGE [key] [start] [end]` - Retrieves a substring from a value.
- `SETRANGE [key] [offset] [value]` - Overwrites part of a string starting at the specified offset.
- `KEYS [pattern]` - Returns all keys matching a pattern.

## JSON Commands
- `JSON.SET [key] [path] [value]` - Sets a JSON value at the specified path.
- `JSON.GET [key] [path]` - Gets the JSON value at the specified path.
- `JSON.DEL [key] [path]` - Deletes the JSON value at the specified path.
- `JSON.ARRAPPEND [key] [path] [value]` - Appends values to a JSON array.<br>
`JSON.SET mykey . '{"name":"John", "age":30}'`<br>
`JSON.GET mykey .name`<br>
  `JSON.SET key . '{ "key": "value1", "nested": { "key2": "value2" }, "arr": ["first", "second", "third"] }'`<br>
  `JSON.GET key .`<br>
  `JSON.arrappend key .arr "hello, world"`<br>
## List Commands
- `LPUSH [key] [value]` - Prepends a value to a list.
- `RPUSH [key] [value]` - Appends a value to a list.
- `LPOP [key]` - Removes and returns the first element of a list.
- `RPOP [key]` - Removes and returns the last element of a list.
- `LLEN [key]` - Gets the length of a list.
- `LRANGE [key] [start] [stop]` - Gets a range of elements from a list.

## Set Commands
- `SADD [key] [value]` - Adds one or more members to a set.
- `SMEMBERS [key]` - Returns all members of a set.
- `SREM [key] [value]` - Removes one or more members from a set.
- `SISMEMBER [key] [value]` - Checks if a value is a member of a set.

## Hash Commands
- `HSET [key] [field] [value]` - Sets a field in a hash.
- `HGET [key] [field]` - Gets the value of a field in a hash.
- `HMSET [key] [field-value pairs]` - Sets multiple fields in a hash.
- `HGETALL [key]` - Gets all fields and values in a hash.
- `HDEL [key] [field]` - Deletes a field from a hash.
- `HEXISTS [key] [field]` - Checks if a field exists in a hash.

## Sorted Set Commands
- `ZADD [key] [score] [value]` - Adds a member with a score to a sorted set.
- `ZRANGE [key] [start] [stop]` - Returns a range of members by index.
- `ZRANK [key] [member]` - Gets the rank of a member.
- `ZREM [key] [member]` - Removes a member from a sorted set.
- `ZRANGEBYSCORE [key] [min] [max]` - Returns members within a score range.

## Stream Commands
- `XADD [key] [id] [field-value pairs]` - Adds an entry to a stream.
- `XREAD [key] [id]` - Reads entries from a stream.
- `XRANGE [key] [start] [end]` - Gets entries from a range.
- `XLEN [key]` - Gets the number of entries in a stream.
- `XGROUP [subcommand]` - Manages consumer groups.
- `XREADGROUP [group] [consumer] [key]` - Reads entries as part of a consumer group.
- `XACK [key] [group] [id]` - Acknowledges entries in a consumer group.

## Geospatial Commands
- `GEOADD [key] [longitude] [latitude] [member]` - Adds a geospatial item.
- `GEODIST [key] [member1] [member2] [*unit]` - Gets the distance between two members.
- `GEORADIUS [key] [longitude] [latitude] [radius] [*unit]` - Finds members within a radius.

## Bitmap Commands
- `SETBIT [key] [offset] [value]` - Sets or clears the bit at a given offset.
- `GETBIT [key] [offset]` - Gets the bit at a given offset.
- `BITCOUNT [key]` - Counts the number of set bits.
- `BITOP [operation] [destkey] [key...]` - Performs bitwise operations.

## Bit Field Commands
- `BITFIELD [key] SET [type] [offset] [value]` - Sets a value in a bit field.
- `BITFIELD [key] GET [type] [offset]` - Gets a value from a bit field.
- `BITFIELD [key] INCRBY [type] [offset] [increment]` - Increments a value in a bit field.

## HyperLogLog Commands
- `PFADD [key] [element]` - Adds elements to a HyperLogLog.
- `PFMERGE [destkey] [sourcekeys...]` - Merges multiple HyperLogLogs.
- `PFCOUNT [key]` - Gets the approximate cardinality.

## Time Series Commands
- `TS.CREATE [key]` - Creates a time series.
- `TS.ADD [key] [timestamp] [value]` - Adds a data point to a time series.
- `TS.RANGE [key] [start] [end]` - Queries a range of data points.
- `TS.GET [key]` - Gets the latest data point.

## Pub/Sub Commands
- `SUBSCRIBE [channel]` - Subscribes to a channel.
- `UNSUBSCRIBE [channel]` - Unsubscribes from a channel.
- `PUBLISH [channel] [message]` - Publishes a message to a channel.

## Vector Commands
- `VECTOR.SET [key] [vector]` - Stores a vector.
- `VECTOR.GET [key]` - Retrieves a vector.
- `VECTOR.SEARCH [key] [query]` - Searches for vectors similar to the query.
