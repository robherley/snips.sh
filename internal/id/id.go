package id

import "github.com/jaevor/go-nanoid"

const (
	// IDLength is the length of the generated ID.
	// About ~17 years of IDs in order to have a 1% chance of collision
	IDLength = 10
)

var idgen func() string

func init() {
	var err error
	idgen, err = nanoid.Standard(IDLength)
	if err != nil {
		panic(err)
	}
}

func New() string {
	return idgen()
}
