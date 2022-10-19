package slices

func Contains[T comparable](inSliceOfItems []T, lookupForItem T) bool {
	for _, item := range inSliceOfItems {
		if item == lookupForItem {
			return true
		}
	}
	return false
}
