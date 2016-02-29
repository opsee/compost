package main

import (
	"github.com/opsee/compost/composter"
	"github.com/opsee/compost/resolver"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	resolver, err := resolver.NewClient(resolver.ClientConfig{
		Bartnet:  "https://bartnet.in.opsee.com",
		Beavis:   "https://beavis.in.opsee.com",
		Spanx:    "spanx.in.opsee.com:8443",
		Vape:     "vape.in.opsee.com:443",
		Keelhaul: "keelhaul.in.opsee.com:443",
	})

	if err != nil {
		log.Fatal(err)
	}

	composter := composter.New(resolver)
	composter.StartHTTP(
		mustEnvString("COMPOST_ADDRESS"),
	)
}

func mustEnvString(envVar string) string {
	out := os.Getenv(envVar)
	if out == "" {
		log.Fatal(envVar, "must be set")
	}
	return out
}
