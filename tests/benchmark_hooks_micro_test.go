package tests

import (
	"testing"
)

type HookInterface interface {
	Foo()
}

type StructWithHook struct{}

func (s *StructWithHook) Foo() {}

type StructWithoutHook struct{}

func BenchmarkTypeAssertion_Success(b *testing.B) {
	s := &StructWithHook{}
	var v any = s
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if h, ok := v.(HookInterface); ok {
			h.Foo()
		}
	}
}

func BenchmarkTypeAssertion_Failure(b *testing.B) {
	s := &StructWithoutHook{}
	var v any = s
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if h, ok := v.(HookInterface); ok {
			h.Foo()
		}
	}
}

func BenchmarkBoolCheck_Success(b *testing.B) {
	s := &StructWithHook{}
	var v any = s
	hasHook := true
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if hasHook {
			if h, ok := v.(HookInterface); ok {
				h.Foo()
			}
		}
	}
}

func BenchmarkBoolCheck_Failure(b *testing.B) {
	// s := &StructWithoutHook{}
	// var v any = s
	hasHook := false
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if hasHook {
			// This block is skipped
		}
	}
}
