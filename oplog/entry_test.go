package oplog

import (
    "fmt"
    "testing"
)

func TestNewEntry(t *testing.T) {
    e := NewEntry("some entry")    
    fmt.Printf("%s", e.Payload)
}