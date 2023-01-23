package id

import (
	"time"

	"github.com/teris-io/shortid"
)

var _sid *shortid.Shortid

func init() {
	sid, err := shortid.New(0, shortid.DefaultABC, uint64(time.Now().Unix()))
	if err != nil {
		panic(err)
	}

	_sid = sid
}

func Generate() (string, error) {
	return _sid.Generate()
}

func MustGenerate() string {
	return _sid.MustGenerate()
}
