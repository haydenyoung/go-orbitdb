package oplog

type Log struct {
	Entries map[string]string
}

func NewLog() *Log {
	l := Log{}
	l.Entries = make(map[string]string)
	return &l
}

func Append(l Log, data string) {
	hash := "0"
	// bytes := []uint8(data)
	l.Entries[hash] = data
}
