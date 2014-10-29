package cache

import (
	"encoding/gob"
)

func init() {
	gob.Register(b{})
	gob.Register(i{})
	gob.Register(is{})
	gob.Register(s{})
}
