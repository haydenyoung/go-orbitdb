package oplog

type Clock struct {
	ID   string `json:"id"`
	Time int    `json:"time"`
}

func NewClock(id string, time int) Clock {
	return Clock{
		ID:   id,
		Time: time,
	}
}

func CompareClocks(a Clock, b Clock) (res int) {
	dist := a.Time - b.Time
	res = dist

	if dist == 0 && a.ID != b.ID {
		if a.ID < b.ID {
			res = -1
		} else {
			res = 1
		}
	}

	return
}

func TickClock(c Clock) Clock {
	return Clock{ID: c.ID, Time: c.Time + 1}
}

func (c *Clock) Tick() {
	c.Time += 1
}
