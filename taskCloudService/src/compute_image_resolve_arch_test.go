package main

import "testing"

func TestInferInstanceArchitecture(t *testing.T) {
	cases := []struct {
		instanceType string
		want         string
	}{
		{"", ""},
		{"ecs.g7.xlarge", "x86_64"},
		{"ecs.r6.xlarge", "x86_64"},
		{"ecs.r6a.xlarge", "x86_64"},
		{"ecs.r6r.xlarge", "arm64"},
		{"ecs.g6r.large", "arm64"},
		{"ecs.c6r.xlarge", "arm64"},
		{"ecs.g8y.xlarge", "arm64"},
		{"ecs.c8y.large", "arm64"},
		{"ecs.r8y.large", "arm64"},
		{"ecs.c6r.xlarge", "arm64"},
		{"n2-standard-4-arm", "arm64"},
	}
	for _, tc := range cases {
		got := inferInstanceArchitecture(tc.instanceType)
		if got != tc.want {
			t.Errorf("inferInstanceArchitecture(%q)=%q want %q", tc.instanceType, got, tc.want)
		}
	}
}
