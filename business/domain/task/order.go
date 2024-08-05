package task

import (
	"fmt"
	"strings"
)

// Direction represents direction ASC and DESC in ordering rows.
type Direction int

const (
	DirectionASC = iota
	DirectionDESC
)

var directionNames = [...]string{"ASC", "DESC"}

// String implements the stringer interface.
func (d Direction) String() string {
	if d < DirectionASC || d > DirectionDESC {
		return "UNKNOWN"
	}
	return directionNames[d]
}

// ParseDirection creates a direction from a string or return possible errors.
func ParseDirection(dir string) (Direction, error) {
	dir = strings.TrimSpace(dir)

	for i, d := range directionNames {
		if strings.ToUpper(dir) == d {
			return Direction(i), nil
		}
	}
	return Direction(-1), fmt.Errorf("%q is invalid direction", dir)
}

// Field represents the name of fields that client can apply ordering.
type Field int

const (
	FieldCommand Field = iota
	FieldStatus
	FieldCreatedAt
	FieldScheduledAt
	FieldID
)

// these are general names, store layer must map these to real columns in db.
var fieldNames = [...]string{"command", "status", "createdAt", "scheduledAt", "ID"}

// String implements stringer interface.
func (f Field) String() string {
	if f < FieldCommand || f > FieldScheduledAt {
		return "UNKNOWN"
	}
	return fieldNames[f]
}

// ParseField creates a field from a string also returns possible errors.
func ParseField(field string) (Field, error) {
	field = strings.TrimSpace(field)

	for i, f := range fieldNames {
		if strings.ToLower(field) == f {
			return Field(i), nil
		}
	}
	return Field(-1), fmt.Errorf("%q, ivalid field name", field)
}

// OrderBy represents the field and direction to order based on it
type OrderBy struct {
	Field     Field
	Direction Direction
}
