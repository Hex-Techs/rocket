package tools

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestContainsString(t *testing.T) {
	type args struct {
		slice []string
		s     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "test01", args: args{slice: []string{"a"}, s: "a"}, want: true},
		{name: "test02", args: args{slice: []string{"a"}, s: "b"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsString(tt.args.slice, tt.args.s); got != tt.want {
				t.Errorf("ContainsString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveString(t *testing.T) {
	type args struct {
		slice []string
		s     string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{name: "test", args: args{slice: []string{"a", "b"}, s: "a"}, want: []string{"b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveString(tt.args.slice, tt.args.s); !cmp.Equal(got, tt.want) {
				t.Errorf("RemoveString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRandomStr(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantLen int
	}{
		{
			"length of -1",
			-1,
			0,
		},
		{
			"length of 0",
			0,
			0,
		},
		{
			"length of 7",
			7,
			7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RandomStr(tt.length); len(got) != tt.wantLen {
				t.Errorf("RandomStr() = %v, want %v", got, tt.wantLen)
			}
		})
	}
}
