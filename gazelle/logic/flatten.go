package logic

// Merges any nested And/Or expressions and returns the flattened tree.
// Mconfig tends to nest AND/OR identifiers with only 2 arguments, we can use
// flattening to simplify them where possible.
func Flatten(expr Expr) Expr {
	switch expr := expr.(type) {
	case *And:
		for hasAnd := true; hasAnd; {
			flatValues := []Expr{}
			hasAnd = false
			for _, current := range expr.Values {
				switch child := current.(type) {
				case *And:
					hasAnd = true // If we are extracting any child values, we must check again
					flatValues = append(flatValues, child.Values...)
				default:
					flatValues = append(flatValues, current)
				}
			}

			expr.Values = flatValues
		}

		for idx, current := range expr.Values {
			expr.Values[idx] = Flatten(current)
		}

	case *Or:
		for hasOr := true; hasOr; {
			flatValues := []Expr{}
			hasOr = false
			for _, current := range expr.Values {
				switch child := current.(type) {
				case *Or:
					hasOr = true // If we are extracting any child values, we must check again
					flatValues = append(flatValues, child.Values...)
				default:
					flatValues = append(flatValues, current)
				}
			}
			expr.Values = flatValues
		}

		for idx, current := range expr.Values {
			expr.Values[idx] = Flatten(current)
		}
	case *Not:
		expr.Value = Flatten(expr.Value)
	}

	return expr
}
