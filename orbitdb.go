package main

import (
	"fmt"
	"orbitdb/go-orbitdb/oplog"
)

func main() {
    var c = oplog.Clock{Id: "a", Time: 1}
    var newClock = oplog.TickClock(c)
	fmt.Printf("%d", newClock.Time)
}