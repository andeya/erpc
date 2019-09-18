package goutil

import "time"

// KVData key-value data
type KVData map[string]interface{}

// Get returns the value for the given key, ie: (value, true).
// If the value does not exists it returns (nil, false)
func (k KVData) Get(key string) (value interface{}, exists bool) {
	value, exists = k[key]
	return
}

// MustGet returns the value for the given key if it exists, otherwise it panics.
func (k KVData) MustGet(key string) interface{} {
	if value, exists := k.Get(key); exists {
		return value
	}
	panic("Key \"" + key + "\" does not exist")
}

// GetString returns the value associated with the key as a string.
func (k KVData) GetString(key string) (s string) {
	if val, ok := k.Get(key); ok && val != nil {
		s, _ = val.(string)
	}
	return
}

// GetBool returns the value associated with the key as a boolean.
func (k KVData) GetBool(key string) (b bool) {
	if val, ok := k.Get(key); ok && val != nil {
		b, _ = val.(bool)
	}
	return
}

// GetInt returns the value associated with the key as an integer.
func (k KVData) GetInt(key string) (i int) {
	if val, ok := k.Get(key); ok && val != nil {
		i, _ = val.(int)
	}
	return
}

// GetInt64 returns the value associated with the key as an integer.
func (k KVData) GetInt64(key string) (i64 int64) {
	if val, ok := k.Get(key); ok && val != nil {
		i64, _ = val.(int64)
	}
	return
}

// GetFloat64 returns the value associated with the key as a float64.
func (k KVData) GetFloat64(key string) (f64 float64) {
	if val, ok := k.Get(key); ok && val != nil {
		f64, _ = val.(float64)
	}
	return
}

// GetTime returns the value associated with the key as time.
func (k KVData) GetTime(key string) (t time.Time) {
	if val, ok := k.Get(key); ok && val != nil {
		t, _ = val.(time.Time)
	}
	return
}

// GetDuration returns the value associated with the key as a duration.
func (k KVData) GetDuration(key string) (d time.Duration) {
	if val, ok := k.Get(key); ok && val != nil {
		d, _ = val.(time.Duration)
	}
	return
}

// GetStringSlice returns the value associated with the key as a slice of strings.
func (k KVData) GetStringSlice(key string) (ss []string) {
	if val, ok := k.Get(key); ok && val != nil {
		ss, _ = val.([]string)
	}
	return
}

// GetStringMap returns the value associated with the key as a map of interfaces.
func (k KVData) GetStringMap(key string) (sm map[string]interface{}) {
	if val, ok := k.Get(key); ok && val != nil {
		sm, _ = val.(map[string]interface{})
	}
	return
}

// GetStringMapString returns the value associated with the key as a map of strings.
func (k KVData) GetStringMapString(key string) (sms map[string]string) {
	if val, ok := k.Get(key); ok && val != nil {
		sms, _ = val.(map[string]string)
	}
	return
}

// GetStringMapStringSlice returns the value associated with the key as a map to a slice of strings.
func (k KVData) GetStringMapStringSlice(key string) (smss map[string][]string) {
	if val, ok := k.Get(key); ok && val != nil {
		smss, _ = val.(map[string][]string)
	}
	return
}
