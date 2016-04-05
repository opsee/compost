package composter

import (
	"bytes"
	"github.com/opsee/compost/resolver"
	"github.com/opsee/vaper"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	bearerToke = "eyJhbGciOiJBMTI4R0NNS1ciLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiI4ZHBsWGkzNzcwamNva0g5IiwidGFnIjoiWjV1aHhzdFBPY3E3dUUyS0lWcHlGdyJ9.IivL8Lsvn14iVZiQVtd_KQ.2-q6fahxJyVRYjui.4i_MJ_fAmcVEex06i_A0dKAJkKBCCpeb4uU9c_zCUSqXrnKamu7UD4Q9NB5BfGTLqK6TB7Zj5nCc4udejcKx9f_bCqcf89Jfm1keCnSE3NGmhihEpynAolFE1YGaIUPUinJMo9TmCLoSSBm9GyzL9Ombkf8I5D3peHoj9r0Y4dcwZMw7OFTZByTQ6b0oMYmrAuGvi85ZZU5ObTO-VbAy6m45XJfb_mFFx2RFliM8Dm61r60FhdrkME0ZcWjtdWo-GqIl-YtWqOVC-n6r-hSHg5g.upEmBJ4IufBcD9X03S3ofg"
)

var (
	testVapeKey = []byte{194, 164, 235, 6, 138, 248, 171, 239, 24, 216, 11, 22, 137, 199, 215, 133}
)

func init() {
	vaper.Init(testVapeKey)
}

func TestAdminAuth(t *testing.T) {
	assert := assert.New(t)
	c := New(&resolver.Client{})

	req, err := http.NewRequest("POST", "http://compost/admin/graphql", bytes.NewBuffer([]byte(`{"query": "{}"}`)))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+bearerToke)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c.router.ServeHTTP(w, req)

	assert.Equal(401, w.Code)
}
