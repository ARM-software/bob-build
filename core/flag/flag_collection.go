package flag

// Array of files as a helper for struct attribute collections
// TODO: add the possibility to tag a group of files.
type Flags []Flag

func (fs Flags) Contains(query Flag) bool {
	for _, f := range fs {
		if f.ToString() == query.ToString() {
			return true
		}
	}
	return false
}

func (fs Flags) AppendIfUnique(f Flag) Flags {
	if !fs.Contains(f) {
		return append(fs, f)
	}
	return fs
}

func (fs Flags) Merge(other Flags) Flags {
	return append(fs, other...)
}

func (fs Flags) Iterate() <-chan Flag {
	c := make(chan Flag)
	go func() {
		for _, f := range fs {
			c <- f
		}
		close(c)
	}()
	return c
}

func (fs Flags) Filtered(predicate func(Flag) bool) (ret Flags) {
	fs.ForEachIf(predicate,
		func(f Flag) {
			ret = append(ret, f)
		})
	return
}

// Sorts the given collection by it's type masked by given mask.
// For example if mask is flag.TypeCC | flag.TypeInclude, the buckets would be:
// TypeUnset
// TypeCC
// TypeInclude
// TypeCC | TypeInclude
func (fs Flags) GroupByType(mask Type) (out Flags) {
	buckets := map[Type]Flags{}

	fs.ForEach(func(f Flag) {
		buckets[f.Type()&mask] = append(buckets[f.Type()&mask], f)
	})

	// Keep the order of the result matching the order of tag declaration.
	// The order of flags within a bucket should be unchanged.
	for tag := TypeUnset; tag <= mask; tag++ {
		if flags, ok := buckets[tag]; ok {
			flags.ForEach(func(f Flag) {
				out = append(out, f)
			})
		}
	}

	return
}

func (fs Flags) IteratePredicate(predicate func(Flag) bool) <-chan Flag {
	c := make(chan Flag)
	go func() {
		for _, f := range fs {
			if predicate(f) {
				c <- f
			}
		}
		close(c)
	}()
	return c
}

func (fs Flags) ForEach(functor func(Flag)) {
	for f := range fs.Iterate() {
		functor(f)
	}
}

func (fs Flags) ForEachIf(predicate func(Flag) bool, functor func(Flag)) {
	for f := range fs.IteratePredicate(predicate) {
		functor(f)
	}
}
