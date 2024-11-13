package oplog

import (
	"testing"
)

func TestClockAIdLessThanClockBId(t *testing.T) {
	c1 := NewClock("a", 1)
	c2 := NewClock("b", 1)

	expected := -1
	actual := CompareClocks(c1, c2)

	if actual != expected {
		t.Errorf("expected '%d' but got '%d'", expected, actual)
	}
}

func TestClockAIdGreaterThanClockBId(t *testing.T) {
	c1 := NewClock("b", 1)
	c2 := NewClock("a", 1)

	expected := 1
	actual := CompareClocks(c1, c2)

	if actual != expected {
		t.Errorf("expected '%d' but got '%d'", expected, actual)
	}
}

func TestClockSame(t *testing.T) {
	c1 := NewClock("a", 1)
	c2 := NewClock("a", 1)

	expected := 0
	actual := CompareClocks(c1, c2)

	if actual != expected {
		t.Errorf("expected '%d' but got '%d'", expected, actual)
	}
}

func TestClockTime1LessThanTime2By1(t *testing.T) {
	c1 := NewClock("a", 1)
	c2 := NewClock("a", 2)

	expected := -1
	actual := CompareClocks(c1, c2)

	if actual != expected {
		t.Errorf("expected '%d' but got '%d'", expected, actual)
	}
}

func TestClockTime1GreaterThanTime2By1(t *testing.T) {
	c1 := NewClock("a", 2)
	c2 := NewClock("a", 1)

	expected := 1
	actual := CompareClocks(c1, c2)

	if actual != expected {
		t.Errorf("expected '%d' but got '%d'", expected, actual)
	}
}

func TestClockTime1LessThanTime2By10(t *testing.T) {
	c1 := NewClock("a", 1)
	c2 := NewClock("a", 11)

	expected := -10
	actual := CompareClocks(c1, c2)

	if actual != expected {
		t.Errorf("expected '%d' but got '%d'", expected, actual)
	}
}

func TestClockTime1GreaterThanTime2By10(t *testing.T) {
	c1 := NewClock("a", 11)
	c2 := NewClock("a", 1)

	expected := 10
	actual := CompareClocks(c1, c2)

	if actual != expected {
		t.Errorf("expected '%d' but got '%d'", expected, actual)
	}
}

func TestTickClock(t *testing.T) {
	c := NewClock("a", 1)

	expected := 2
	newClock := TickClock(c)

	actual := newClock.Time

	if actual != expected {
		t.Errorf("expected '%d' but got '%d'", expected, actual)
	}
}
