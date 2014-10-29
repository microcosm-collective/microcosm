package cache

type b struct {
	V bool
}

type i struct {
	V int64
}

type is struct {
	V []int64
}

type s struct {
	V string
}

// CacheSetBool is a utility function to put a bool into cache
func CacheSetBool(key string, data bool, timeToLive int32) {
	CacheSet(key, b{V: data}, timeToLive)
}

// CacheGetBool is a utility function to get a bool from cache
func CacheGetBool(key string) (bool, bool) {
	if val, ok := CacheGet(key, b{}); ok {
		return val.(b).V, true
	}

	return false, false
}

// CacheSetInt64 is a utility function to put an int64 into cache
func CacheSetInt64(key string, data int64, timeToLive int32) {
	CacheSet(key, i{V: data}, timeToLive)
}

// CacheGetInt64 is a utility function to get an int64 from cache
func CacheGetInt64(key string) (int64, bool) {
	if val, ok := CacheGet(key, i{}); ok {
		return val.(i).V, true
	}

	return 0, false
}

// CacheSetInt64Slice is a utility function to put a slice of int64 into cache
func CacheSetInt64Slice(key string, data []int64, timeToLive int32) {
	CacheSet(key, is{V: data}, timeToLive)
}

// CacheGetInt64Slice is a utility function to get a slice of int64 from cache
func CacheGetInt64Slice(key string) ([]int64, bool) {
	if val, ok := CacheGet(key, is{}); ok {
		return val.(is).V, true
	}

	return []int64{}, false
}

// CacheSetString is a utility function to put a string into cache
func CacheSetString(key string, data string, timeToLive int32) {
	CacheSet(key, s{V: data}, timeToLive)
}

// CacheGetString is a utility function to get a string from cache
func CacheGetString(key string) (string, bool) {
	if val, ok := CacheGet(key, s{}); ok {
		return val.(s).V, true
	}

	return "", false
}
