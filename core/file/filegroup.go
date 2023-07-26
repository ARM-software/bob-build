package file

// Array of files as a helper for struct attribute collections
// TODO: add the possibility to tag a group of files.
type Paths []Path

func (fps Paths) Contains(query Path) bool {
	for _, fp := range fps {
		if fp == query {
			return true
		}
	}
	return false
}

func (fps Paths) AppendIfUnique(fp Path) Paths {
	if !fps.Contains(fp) {
		return append(fps, fp)
	}
	return fps
}

func (fps Paths) Merge(other Paths) Paths {
	return append(fps, other...)
}

func (fps Paths) Iterate() <-chan Path {
	c := make(chan Path)
	go func() {
		for _, fp := range fps {
			c <- fp
		}
		close(c)
	}()
	return c
}

func (fps Paths) IteratePredicate(predicate func(Path) bool) <-chan Path {
	c := make(chan Path)
	go func() {
		for _, fp := range fps {
			if predicate(fp) {
				c <- fp
			}
		}
		close(c)
	}()
	return c
}

func (fps Paths) ForEach(functor func(Path) bool) {
	for fp := range fps.Iterate() {
		if !functor(fp) {
			break
		}
	}
}

func (fps Paths) ForEachIf(predicate func(Path) bool, functor func(Path) bool) {
	for fp := range fps.IteratePredicate(predicate) {
		if !functor(fp) {
			break
		}
	}
}

func (fps Paths) FindSingle(predicate func(Path) bool) (*Path, bool) {
	for fp := range fps.Iterate() {
		if predicate(fp) {
			return &fp, true
		}
	}
	return nil, false
}

func (fs Paths) Filtered(predicate func(Path) bool) (ret Paths) {
	fs.ForEachIf(predicate,
		func(f Path) bool {
			ret = append(ret, f)
			return true
		})
	return
}

func (fs Paths) ToStringSliceIf(predicate func(Path) bool, converter func(Path) string) (ret []string) {
	fs.ForEachIf(predicate,
		func(f Path) bool {
			ret = append(ret, converter(f))
			return true
		})
	return
}

func (fs Paths) ToStringSlice(converter func(Path) string) (ret []string) {
	fs.ForEach(
		func(f Path) bool {
			ret = append(ret, converter(f))
			return true
		})
	return
}
