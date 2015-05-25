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

// SetBool is a utility function to put a bool into cache
func SetBool(key string, data bool, timeToLive int32) {
	Set(key, b{V: data}, timeToLive)
}

// GetBool is a utility function to get a bool from cache
func GetBool(key string) (bool, bool) {
	if val, ok := Get(key, b{}); ok {
		return val.(b).V, true
	}

	return false, false
}

// SetInt64 is a utility function to put an int64 into cache
func SetInt64(key string, data int64, timeToLive int32) {
	Set(key, i{V: data}, timeToLive)
}

// GetInt64 is a utility function to get an int64 from cache
func GetInt64(key string) (int64, bool) {
	if val, ok := Get(key, i{}); ok {
		return val.(i).V, true
	}

	return 0, false
}

// SetInt64Slice is a utility function to put a slice of int64 into cache
func SetInt64Slice(key string, data []int64, timeToLive int32) {
	Set(key, is{V: data}, timeToLive)
}

// GetInt64Slice is a utility function to get a slice of int64 from cache
func GetInt64Slice(key string) ([]int64, bool) {
	if val, ok := Get(key, is{}); ok {
		return val.(is).V, true
	}

	return []int64{}, false
}

// SetString is a utility function to put a string into cache
func SetString(key string, data string, timeToLive int32) {
	Set(key, s{V: data}, timeToLive)
}

// GetString is a utility function to get a string from cache
func GetString(key string) (string, bool) {
	if val, ok := Get(key, s{}); ok {
		return val.(s).V, true
	}

	return "", false
}
