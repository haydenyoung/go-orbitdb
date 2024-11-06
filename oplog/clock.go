package oplog

type Clock struct {
	id   string
	time int
}

func NewClock(id string, time int) *Clock {
	c := Clock{id: id, time: time}
	return &c
}

func CompareClocks(a Clock, b Clock) (res int) {
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

func TickClock(c Clock) Clock {
	return Clock{id: c.id, time: c.time + 1}
}
