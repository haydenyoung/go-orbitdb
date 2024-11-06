package oplog

type log struct {
	Entries map[string]string
}

func NewLog() *log {
	l := log{}
	l.Entries = make(map[string]string)
	return &l
}

func Append(l log, data string) {
	hash := "0"
	// bytes := []uint8(data)
	l.Entries[hash] = data
}
