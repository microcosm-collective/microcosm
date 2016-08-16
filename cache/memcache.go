package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/golang/glog"
)

// Maintains a list of constants that determine the type of content held in a
// key. A single ID may have multiple bits of data, i.e.
//   key_1 = 'detail for ID 1'
//   key_2 = 'summary for ID 1'
// This allows us to nuke item 1 from cache and to purge the detail and summary
// for the item at the same time
const (
	CacheDetail     int = 1
	CacheSummary    int = 2
	CacheTitle      int = 3
	CacheItem       int = 4
	CacheDomain     int = 5
	CacheSubdomain  int = 6
	CacheUser       int = 7
	CacheProfileIds int = 8
	CacheCounts     int = 9
	CacheOptions    int = 10
	CacheRootID     int = 11 // Used only by the site type
	CacheBreadcrumb int = 12
	CacheAttributes int = 13
)

var (
	mc      *memcache.Client
	enabled bool
)

// InitCache creates the cache client and enables the cache functions
// within this package. It is the responsibility of whatever has the values for
// this function (usually main.go shortly after reading the config file) to call
// this.
func InitCache(host string, port int64) {
	mc = memcache.New(fmt.Sprintf("%s:%d", host, port))
	enabled = true
}

// Set puts the given interface into the cache
func Set(key string, data interface{}, timeToLive int32) {
	if !enabled {
		return
	}

	// Encode the data for serialisation in memcache
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(&data)
	if err != nil {
		glog.Errorf("enc.Encode(&data) %+v", err)
		return
	}

	err = mc.Set(
		&memcache.Item{
			Key:        key,
			Value:      buf.Bytes(),
			Expiration: timeToLive, // time in seconds
		},
	)
	if err != nil {
		glog.Errorf("mc.Set() %+v", err)
		return
	}
}

// Get gets the data for the given key, if the data is in the cache
func Get(key string, dst interface{}) (interface{}, bool) {
	if !enabled {
		return nil, false
	}

	item, err := mc.Get(key)
	if err != nil {
		// Cache misses are expected, but other errors are logged.
		if err != memcache.ErrCacheMiss {
			glog.Warningf("mc.Get(key) %+v", err)
		}
		return nil, false
	}

	var buf bytes.Buffer
	buf.Write(item.Value)
	dec := gob.NewDecoder(&buf)
	err = dec.Decode(&dst)
	if err != nil {
		glog.Errorf("dec.Decode(&dst) %+v", err)
		return nil, false
	}

	return dst, true
}

// Delete removes items matching the given key from the cache, if it is in
// the cache
func Delete(key string) {
	if !enabled {
		return
	}

	err := mc.Delete(key)
	if err != nil && err != memcache.ErrCacheMiss {
		glog.Warningf("mc.Delete(key) %+v", err)
	}
}
