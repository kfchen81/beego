package cache

type Cache interface {
	// Set a key with value to the cache. Returns true if an item was
	// evicted.
	Set(key, value interface{}) bool

	// Get an item from the cache by key. Returns the value if it exists,
	// and a bool stating whether or not it existed.
	Get(key interface{}) (interface{}, bool)

	// Keys returns a slice of all the keys in the cache
	Keys() []interface{}

	// Len returns the number of items present in the cache
	Len() int

	// Cap returns the total number of items the cache can retain
	Cap() int

	// Purge removes all items from the cache
	Purge()

	// Del deletes an item from the cache by key. Returns if an item was
	// actually deleted.
	Del(key interface{}) bool

	// Del deletes an item from the cache by value. Returns if an item was
	// actually deleted.
	DelByValue(value interface{}) bool
}
