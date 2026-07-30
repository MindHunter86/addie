package utils

type BoltBucket uint16

// type BoltColumn uint16

const (
	BBPayloadFPrints BoltBucket = iota
)

var BoltBuckets = map[BoltBucket]string{
	BBPayloadFPrints: "pfprints",
}
