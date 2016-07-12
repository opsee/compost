package resolver

import (
	"fmt"
	"github.com/opsee/basic/schema"
	"golang.org/x/net/context"
	"testing"
)

func TestResolveChecks(t *testing.T) {
	// skipping bc this must run on the vpn
	t.SkipNow()
	resolver, err := NewClient(ClientConfig{
		SkipVerify: false,
		Bartnet:    "https://bartnet.in.opsee.com",
		Beavis:     "https://beavis.in.opsee.com",
		Spanx:      "spanx.in.opsee.com:8443",
		Cats:       "cats.in.opsee.com:443",
		Keelhaul:   "keelhaul.in.opsee.com:443",
		Bezos:      "opsee.local:9104",
		Hugs:       "https://hugs.in.opsee.com",
	})

	if err != nil {
		t.Fatal(err)
	}

	user := &schema.User{
		Id:         int32(7),
		CustomerId: "140c5346-5d57-11e5-9947-9f9fcf62725e",
		Email:      "computer@markmart.in",
		Active:     true,
		Verified:   true,
	}

	checks, err := resolver.ListChecks(context.Background(), user, "")
	if err != nil {
		t.Fatal(err)
	}

	for _, check := range checks {
		fmt.Printf("%#v\n", check.Notifications)
	}
}
