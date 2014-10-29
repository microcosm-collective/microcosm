package resolver

import (
	"encoding/gob"
)

func init() {
	// Required by the cache stuff
	gob.Register(Origin{})
}
