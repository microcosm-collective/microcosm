/*
Package cache provides an interface to cache items. It should not be of any
concern to the callee where this cache is, simply that the cache exists and will
speed things up.

Eventual consistency of the cached items is promised, but nothing more.
*/
package cache
