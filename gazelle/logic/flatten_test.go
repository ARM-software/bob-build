package logic

import (
	"reflect"
	"testing"
)

var flattenTests = []struct {
	input    Expr
	expected Expr
}{
	// Base case
	{
		&Identifier{"A"},
		&Identifier{"A"},
	},
	// Nested And
	{
		&And{
			[]Expr{
				&And{
					[]Expr{
						&Identifier{Value: "A"},
						&Identifier{Value: "B"},
						&And{
							[]Expr{
								&Identifier{Value: "C"},
								&Identifier{Value: "D"},
							},
						},
					},
				},
				&Identifier{Value: "E"},
				&Identifier{Value: "F"},
			},
		},
		&And{
			[]Expr{
				&Identifier{Value: "A"},
				&Identifier{Value: "B"},
				&Identifier{Value: "C"},
				&Identifier{Value: "D"},
				&Identifier{Value: "E"},
				&Identifier{Value: "F"},
			},
		},
	},
	// Nested Or
	{
		&Or{
			[]Expr{
				&Or{
					[]Expr{
						&Identifier{Value: "A"},
						&Identifier{Value: "B"},
						&Or{
							[]Expr{
								&Identifier{Value: "C"},
								&Identifier{Value: "D"},
							},
						},
					},
				},
				&Identifier{Value: "E"},
				&Identifier{Value: "F"},
			},
		},
		&Or{
			[]Expr{
				&Identifier{Value: "A"},
				&Identifier{Value: "B"},
				&Identifier{Value: "C"},
				&Identifier{Value: "D"},
				&Identifier{Value: "E"},
				&Identifier{Value: "F"},
			},
		},
	},
	// Basic mixed mode, no flattening possible
	{
		&Or{
			[]Expr{
				&And{
					[]Expr{
						&Identifier{Value: "A"},
						&Identifier{Value: "B"},
						&Or{
							[]Expr{
								&Identifier{Value: "C"},
								&Identifier{Value: "D"},
							},
						},
					},
				},
				&Identifier{Value: "E"},
				&Identifier{Value: "F"},
			},
		},
		&Or{
			[]Expr{
				&And{
					[]Expr{
						&Identifier{Value: "A"},
						&Identifier{Value: "B"},
						&Or{
							[]Expr{
								&Identifier{Value: "C"},
								&Identifier{Value: "D"},
							},
						},
					},
				},
				&Identifier{Value: "E"},
				&Identifier{Value: "F"},
			},
		},
	},
	// Basic mixed mode with flattening
	{
		&Or{
			[]Expr{
				&Or{
					[]Expr{
						&Identifier{Value: "A"},
						&Identifier{Value: "B"},
						&And{
							[]Expr{
								&Identifier{Value: "C"},
								&Identifier{Value: "D"},
							},
						},
					},
				},
				&Identifier{Value: "E"},
				&Identifier{Value: "F"},
			},
		},
		&Or{
			[]Expr{
				&Identifier{Value: "A"},
				&Identifier{Value: "B"},
				&And{
					[]Expr{
						&Identifier{Value: "C"},
						&Identifier{Value: "D"},
					},
				},
				&Identifier{Value: "E"},
				&Identifier{Value: "F"},
			},
		},
	},
}

func Test_FlattenNestedAnd(t *testing.T) {
	for _, tt := range flattenTests {
		actual := Flatten(tt.input)
		if !reflect.DeepEqual(actual, tt.expected) {
			t.Errorf("Flatten failed! Input:'%v' actual:'%v' expected:'%v'", tt.input, actual, tt.expected)
		}
	}
}
