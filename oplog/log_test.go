package oplog

import (
	"fmt"
	"testing"
)

func TestAddEntryToLog(t *testing.T) {
	l := NewLog()
	Append(*l, "some entry")
	fmt.Printf("%d", len(l.Entries))
}
