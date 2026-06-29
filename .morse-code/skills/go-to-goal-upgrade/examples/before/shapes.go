// Package shapes is a small, self-contained Go package used as the dogfood
// example for the go->goal upgrade skill. It deliberately exercises all four
// manual idioms plus a goal-fix-able propagation shape:
//
//   1. an iota const block (Kind)            -> goal enum
//   2. a method on the would-be-enum (Label) -> free label function
//   3. a type-switch over a closed interface -> sealed interface + match
//   4. an exported fallible (T, error)       -> Result/?
package shapes

import (
	"errors"
	"fmt"
)

// Kind classifies a diagnostic. Pure tag set, no numeric/wire/ordering use ->
// converts to a goal enum.
type Kind int

const (
	Info Kind = iota
	Warn
	Fatal
)

// Label is a method on the would-be-enum -> becomes a free function in goal
// (enums lower to interfaces; Go forbids methods on interface types).
func (k Kind) Label() string {
	switch k {
	case Info:
		return "info"
	case Warn:
		return "warn"
	default:
		return "fatal"
	}
}

// Shape is a closed set of two concrete shapes -> seal it and convert the
// type-switch in Describe to an exhaustive match.
type Shape interface {
	Area() float64
}

// Circle implements Shape.
type Circle struct {
	R float64
}

func (c *Circle) Area() float64 { return 3.14159 * c.R * c.R }

// Rect implements Shape.
type Rect struct {
	W, H float64
}

func (r *Rect) Area() float64 { return r.W * r.H }

// Describe type-switches over the closed Shape set -> exhaustive match in goal.
func Describe(s Shape) string {
	switch v := s.(type) {
	case *Circle:
		return fmt.Sprintf("circle r=%g", v.R)
	case *Rect:
		return fmt.Sprintf("rect %gx%g", v.W, v.H)
	default:
		return "unknown"
	}
}

// ParseDim is a pure-propagation fallible (float64, error) -> Result[float64, error].
func ParseDim(s string) (float64, error) {
	if s == "" {
		return 0, errors.New("empty dimension")
	}
	return float64(len(s)), nil
}

// MakeRect chains two fallible calls with manual if-err propagation -> Result + ?.
func MakeRect(w, h string) (*Rect, error) {
	ww, err := ParseDim(w)
	if err != nil {
		return nil, err
	}
	hh, err := ParseDim(h)
	if err != nil {
		return nil, err
	}
	return &Rect{W: ww, H: hh}, nil
}
