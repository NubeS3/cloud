package cassandra

//func ErrLog(msg string, t string) {
//	now := time.Now()
//	query := session.
//		Query(`INSERT INTO errors_log (type, message, time) VALUES (?, ?, ?) IF NOT EXISTS`,
//			t,
//			msg,
//			now,
//		)
//	if err := query.Exec(); err != nil {
//		return
//	}
//
//	return
//}

func ErrLog(msg string, t string) {
	println(msg, t)
	return
}
