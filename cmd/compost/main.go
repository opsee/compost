package main

import (
	"github.com/opsee/compost/composter"
	"github.com/opsee/compost/resolver"
	"github.com/opsee/vaper"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
)

func main() {
	key, err := ioutil.ReadFile(mustEnvString("COMPOST_VAPE_KEYFILE"))
	if err != nil {
		log.Fatal("Unable to read vape key: ", err)
	}
	vaper.Init(key)

	// for local dev only
	skipVerify := os.Getenv("COMPOST_SKIP_VERIFY")

	resolver, err := resolver.NewClient(resolver.ClientConfig{
		SkipVerify: skipVerify == "true",
		Bartnet:    "https://bartnet.in.opsee.com",
		Beavis:     "https://beavis.in.opsee.com",
		Spanx:      "spanx.in.opsee.com:8443",
		Vape:       "vape.in.opsee.com:443",
		Keelhaul:   "keelhaul.in.opsee.com:443",
		Bezos:      "opsee.local:9104",
		Hugs:       "https://hugs.in.opsee.com",
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
