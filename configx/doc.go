// Package configx provides multi-source configuration loading with typed
// accessors and optional live-reload.
//
// Sources are merged in priority order (highest first):
//  1. Environment variables (WithEnv)
//  2. JSON/YAML config file (WithFile)
//  3. Defaults (WithDefaults)
//
// # Quick Start
//
//	cfg, err := configx.Load(
//	    configx.WithFile("config/app.json"),
//	    configx.WithEnv("APP"),
//	    configx.WithDefaults(map[string]any{
//	        "server.port": 8080,
//	    }),
//	    configx.WithRequired("database.dsn"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	port := cfg.Int("server.port")
//	dsn  := cfg.String("database.dsn")
package configx
