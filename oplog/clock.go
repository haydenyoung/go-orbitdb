package oplog

type clock struct {
	id   string
	time int
}

func NewClock(id string, time int) *clock {
	c := clock{id: id, time: time}
	return &c
}

func CompareClocks(a clock, b clock) (res int) {
	dist := a.time - b.time
	res = dist

	if dist == 0 && a.id != b.id {
		if a.id < b.id {
			res = -1
		} else {
			res = 1
		}
	}

	return
}

func TickClock(c clock) clock {
	return clock{id: c.id, time: c.time + 1}
}
