package configx_test

import (
	"fmt"

	"github.com/santekno/sdk/configx"
)

func ExampleLoad() {
	cfg, err := configx.Load(
		configx.WithDefaults(map[string]any{
			"server.port": 8080,
			"app.name":    "demo",
		}),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(cfg.Int("server.port"))
	fmt.Println(cfg.String("app.name"))
	// Output:
	// 8080
	// demo
}

func ExampleConfig_StringDefault() {
	cfg, _ := configx.Load()
	fmt.Println(cfg.StringDefault("missing.key", "default-value"))
	// Output:
	// default-value
}

func ExampleConfig_IntDefault() {
	cfg, _ := configx.Load()
	fmt.Println(cfg.IntDefault("missing.port", 3000))
	// Output:
	// 3000
}
