package collections

type Group[I any, K comparable] struct {
	Key     K
	Members []I
}

// GroupBy partitions items into groups by applying keyFn to each element.
// Groups are returned in the order their keys first appear in items.
func GroupBy[I any, K comparable](items []I, keyFn func(I) K) []Group[I, K] {
	order, groupMap := groupBy(items, keyFn)

	groups := make([]Group[I, K], len(order))

	for i, key := range order {
		groups[i] = Group[I, K]{
			Key:     key,
			Members: groupMap[key],
		}
	}

	return groups
}

func groupBy[I any, K comparable](slice []I, keyFn func(I) K) ([]K, map[K][]I) {
	order := []K{}
	groups := map[K][]I{}

	for _, item := range slice {
		key := keyFn(item)
		if _, exists := groups[key]; !exists {
			order = append(order, key)
		}
		groups[key] = append(groups[key], item)
	}

	return order, groups
}
