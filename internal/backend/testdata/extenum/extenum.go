// Package extenum is a fixture foreign Go package imported by the backend's
// cross-package enum-match test and corpus case. It carries the generated §8.1
// sum encoding of a tag-only enum `Light { On Off }` — the marker interface plus
// one variant struct and marker method per variant — exactly as the goal backend
// would emit it for `enum Light { On Off }`. It lives under testdata so the go
// tool never builds it as part of the module.
package extenum

// Light is the §8.1 marker interface for the enum `Light { On Off }`.
type Light interface{ isLight() }

// Light_On is the data-less `On` variant.
type Light_On struct{}

// Light_Off is the data-less `Off` variant.
type Light_Off struct{}

func (Light_On) isLight()  {}
func (Light_Off) isLight() {}
