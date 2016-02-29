package spanx

import (
	"encoding/json"
	"fmt"
	"github.com/opsee/basic/schema"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCredentials(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		creds, err := json.Marshal(map[string]interface{}{
			"Credentials": map[string]string{
				"AccessKeyID":     "hey",
				"SecretAccessKey": "there",
			},
		})

		if err != nil {
			t.Fatal(err)
		}

		fmt.Fprintln(w, string(creds))
	}))

	defer ts.Close()

	user := &schema.User{
		CustomerId: "heyyyy",
	}

	client := New(ts.URL)
	creds, err := client.GetCredentials(user)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "hey", creds.AccessKeyID)

	creds, err = client.PutRole(user, "what", "up")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "there", creds.SecretAccessKey)
}
