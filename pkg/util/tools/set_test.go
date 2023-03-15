package tools

import (
	"testing"
)

type testCase[T comparable] struct {
	name  string
	args  Set[T]
	param Set[T]
	want  Set[T]
}

func runIntersectionCase[T comparable](t *testing.T, cases []testCase[T]) {
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.Intersection(tt.param); !got.Equals(tt.want) {
				t.Errorf("Set.Intersection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_Intersection(t *testing.T) {
	intCase1 := []testCase[int]{
		{
			name: "testInt",
			args: Set[int]{
				m: map[int]EmptyType{
					1: empty,
					2: empty,
				},
			},
			param: Set[int]{
				m: map[int]EmptyType{
					2: empty,
					3: empty,
				},
			},
			want: Set[int]{
				m: map[int]EmptyType{
					2: empty,
				},
			},
		},
	}
	intCase2 := []testCase[int]{
		{
			name: "testInt",
			args: Set[int]{
				m: map[int]EmptyType{
					1: empty,
				},
			},
			param: Set[int]{
				m: map[int]EmptyType{
					3: empty,
				},
			},
			want: Set[int]{
				m: map[int]EmptyType{},
			},
		},
	}
	strCase := []testCase[string]{
		{
			name: "testStr",
			args: Set[string]{
				m: map[string]EmptyType{
					"a": empty,
					"b": empty,
				},
			},
			param: Set[string]{
				m: map[string]EmptyType{
					"b": empty,
					"c": empty,
				},
			},
			want: Set[string]{
				m: map[string]EmptyType{
					"b": empty,
				},
			},
		},
	}
	runIntersectionCase(t, intCase1)
	runIntersectionCase(t, intCase2)
	runIntersectionCase(t, strCase)
}

func runUnionCase[T comparable](t *testing.T, cases []testCase[T]) {
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.Union(tt.param); !got.Equals(tt.want) {
				t.Errorf("Set.Union() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_Union(t *testing.T) {
	intCase := []testCase[int]{
		{
			name: "testInt",
			args: Set[int]{
				m: map[int]EmptyType{
					1: empty,
					2: empty,
				},
			},
			param: Set[int]{
				m: map[int]EmptyType{
					2: empty,
					3: empty,
				},
			},
			want: Set[int]{
				m: map[int]EmptyType{
					1: empty,
					2: empty,
					3: empty,
				},
			},
		},
	}
	strCase := []testCase[string]{
		{
			name: "testStr",
			args: Set[string]{
				m: map[string]EmptyType{
					"a": empty,
					"b": empty,
				},
			},
			param: Set[string]{
				m: map[string]EmptyType{
					"b": empty,
					"c": empty,
				},
			},
			want: Set[string]{
				m: map[string]EmptyType{
					"a": empty,
					"b": empty,
					"c": empty,
				},
			},
		},
	}
	runUnionCase(t, intCase)
	runUnionCase(t, strCase)
}

func runDiffCase[T comparable](t *testing.T, cases []testCase[T]) {
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.Difference(tt.param); !got.Equals(tt.want) {
				t.Errorf("Set.Difference() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_Diff(t *testing.T) {
	intCase := []testCase[int]{
		{
			name: "testInt",
			args: Set[int]{
				m: map[int]EmptyType{
					1: empty,
					2: empty,
				},
			},
			param: Set[int]{
				m: map[int]EmptyType{
					2: empty,
					3: empty,
				},
			},
			want: Set[int]{
				m: map[int]EmptyType{
					1: empty,
				},
			},
		},
	}
	strCase := []testCase[string]{
		{
			name: "testStr",
			args: Set[string]{
				m: map[string]EmptyType{
					"a": empty,
					"b": empty,
				},
			},
			param: Set[string]{
				m: map[string]EmptyType{
					"b": empty,
					"c": empty,
				},
			},
			want: Set[string]{
				m: map[string]EmptyType{
					"a": empty,
				},
			},
		},
	}
	runDiffCase(t, intCase)
	runDiffCase(t, strCase)
}
