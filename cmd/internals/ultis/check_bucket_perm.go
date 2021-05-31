package ultis

func CheckBucketPerm(keyBid, bucketId string) bool {
	return keyBid == "*" || keyBid == bucketId
}
