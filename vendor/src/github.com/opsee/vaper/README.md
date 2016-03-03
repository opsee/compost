vaper
=====

A library to turn structs into JWS or JWE.

Usage
-----

To have struct fields show up in web tokens, tag them with token:"yourtokenfieldname".

```
import (
      "github.com/opsee/vaper"
)

type User struct {
	Id                 int       `token:"id"`
	Email              string    `token:"email"`
	CreatedAt          time.Time `token:"created_at"`
	Admin              bool      `token:"admin"`
	ThisFieldIsIgnored bool
}

user := &User{
  Id: 1,
  Email: "vapin@vape.it",
  CreatedAt: time.Now(),
  Admin: true,
}

now := time.Now()
exp := now.Add(time.Hour)
token := vaper.New(user, "vapin@vape.it", now, exp)
tokenString, err := token.Marshal()

// verify token string
err := vaper.Verify(tokenString)

// reify the token string
decoded, err := vaper.Unmarshal(tokenString)
user := &User{}
err = decoded.Reify(user)
```

TODO
----

*NOTE:* this will be a public project, so watch what you put in here.

A more correct api would be:

```
generator := vaper.New(alg, enc string, exp time.Duration)
token, err := generator.Token(user)
```

- generator as its own struct
- struct tag for subject field
- configurable algorithms
- configurable encryption
- more robust reflection for number types
- marshal into io.Writer
- benchmarking
- godoc
