package vaper

import (
	. "gopkg.in/check.v1"
	"testing"
	"time"
)

type TokenSuite struct{}
type testUser struct {
	Id                 int       `token:"id"`
	Email              string    `token:"email"`
	CreatedAt          time.Time `token:"created_at"`
	Admin              bool      `token:"admin"`
	DumbId             int32     `token:"dumb_id"`
	ThisFieldIsIgnored bool
}

var (
	testVapeKey = []byte{194, 164, 235, 6, 138, 248, 171, 239, 24, 216, 11, 22, 137, 199, 215, 133}
	_           = Suite(&TokenSuite{})
)

func Test(t *testing.T) { TestingT(t) }

func (s *TokenSuite) SetUpTest(c *C) {
	Init(testVapeKey)
}

func (s *TokenSuite) TestNew(c *C) {
	now := time.Now()
	exp := now.Add(time.Hour * 1)
	token := newTestToken(now, exp)

	c.Assert((*token)["exp"], DeepEquals, exp.Unix())
	c.Assert((*token)["ThisFieldIsIgnored"], DeepEquals, nil)
	c.Assert((*token)["email"], DeepEquals, "vapin@vape.it")
	c.Assert((*token)["sub"], DeepEquals, "vapin@vape.it")
}

func (s *TokenSuite) TestReify(c *C) {
	now := time.Now().UTC()
	exp := now.Add(time.Hour * 1)
	token := newTestToken(now, exp)

	tokenString, err := token.Marshal()
	if err != nil {
		c.Fatal(err)
	}

	decoded, err := Unmarshal(tokenString)
	if err != nil {
		c.Fatal(err)
	}

	user := &testUser{}
	decoded.Reify(user)

	c.Assert(user.Id, DeepEquals, 1)
	c.Assert(user.Email, DeepEquals, "vapin@vape.it")
	c.Assert(user.CreatedAt, DeepEquals, now)
	c.Assert(user.Admin, DeepEquals, true)
	c.Assert(user.DumbId, DeepEquals, int32(666))
}

func (s *TokenSuite) TestMarshalUnmarshal(c *C) {
	now := time.Now().UTC()
	exp := now.Add(time.Hour * 1)
	token := newTestToken(now, exp)
	tokenString, err := token.Marshal()
	if err != nil {
		c.Fatal(err)
	}

	decoded, err := Unmarshal(tokenString)
	if err != nil {
		c.Fatal(err)
	}

	c.Assert((*decoded)["exp"], DeepEquals, exp.Unix())
	c.Assert((*decoded)["ThisFieldIsIgnored"], DeepEquals, nil)
	c.Assert((*decoded)["email"], DeepEquals, "vapin@vape.it")
	c.Assert((*decoded)["sub"], DeepEquals, "vapin@vape.it")
}

func (s *TokenSuite) TestVerify(c *C) {
	now := time.Now().UTC()
	exp := now.Add(time.Hour * 1)
	tokenString, err := newTestToken(now, exp).Marshal()
	if err != nil {
		c.Fatal(err)
	}
	c.Assert(Verify(tokenString), IsNil)

	now = time.Now().Add(time.Hour * 1)
	exp = now.Add(time.Hour * 2)
	tokenString, err = newTestToken(now, exp).Marshal()
	if err != nil {
		c.Fatal(err)
	}
	c.Assert(Verify(tokenString), ErrorMatches, ".*issued after now")

	now = time.Now()
	exp = now.Add(time.Hour * -2)
	tokenString, err = newTestToken(now, exp).Marshal()
	if err != nil {
		c.Fatal(err)
	}
	c.Assert(Verify(tokenString), ErrorMatches, ".*expired")
}

func newTestToken(now, exp time.Time) *Token {
	user := &testUser{
		Id:                 1,
		Email:              "vapin@vape.it",
		CreatedAt:          now,
		Admin:              true,
		ThisFieldIsIgnored: true,
		DumbId:             int32(666),
	}

	return New(user, "vapin@vape.it", now, exp)
}
