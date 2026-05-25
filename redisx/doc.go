// Package redisx provides Redis helper utilities built on top of a pluggable
// RedisClient interface. The package is dependency-free: bring your own Redis
// client (e.g. github.com/redis/go-redis/v9) and satisfy the interface.
//
// # Design
//
// redisx defines a minimal [RedisClient] interface covering GET, SET, DEL, and
// SETNX commands. Any Redis client that implements these methods works — including
// the popular github.com/redis/go-redis/v9 package.
//
// # Quick Start
//
//	import (
//	    "github.com/redis/go-redis/v9"         // your choice
//	    "github.com/santekno/sdk/redisx"
//	)
//
//	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	client := redisx.New(rdb)  // rdb satisfies redisx.RedisClient
//
//	var result MyStruct
//	_ = client.GetJSON(ctx, "my:key", &result)
//	_ = client.SetJSON(ctx, "my:key", myObj, time.Hour)
//
//	// Distributed lock
//	unlock, err := client.Lock(ctx, "my:lock", time.Second*10)
//	if err == nil {
//	    defer unlock()
//	}
package redisx
