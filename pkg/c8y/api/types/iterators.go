package types

import "iter"

// ByteProvider represents types that can provide their raw bytes
type ByteProvider interface {
	Bytes() []byte
}

// CollectionIterator represents types that can iterate over a collection
// of ByteProvider items. The actual iterator may yield a concrete type
// that implements ByteProvider (like JSONDoc), which will be converted
// to ByteProvider by the consumer.
type CollectionIterator interface {
	ByteProvider
	IterBytes() iter.Seq[ByteProvider]
}
