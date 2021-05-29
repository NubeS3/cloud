package ultis

import "time"

const zeroTimeStamp = -62135596800

func TimeCheck(t time.Time) bool {
	return t.Unix() != zeroTimeStamp && t.Before(time.Now())
}
