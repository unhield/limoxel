package workspace

import (
	"errors"
	"fmt"
)

var (
	// ErrDuplicateResource indicates a resource with the same ID already exists.
	ErrDuplicateResource = errors.New("workspace: duplicate resource ID")

	// ErrResourceNotFound indicates a requested resource was not found.
	ErrResourceNotFound = errors.New("workspace: resource not found")
)

// Resources represents an immutable collection of Resource objects.
type Resources struct {
	items map[string]*Resource
	order []string
}

// NewResources constructs a new immutable Resources collection from items.
func NewResources(resources ...*Resource) (*Resources, error) {
	itemMap := make(map[string]*Resource, len(resources))
	orderList := make([]string, 0, len(resources))

	for _, res := range resources {
		if res == nil {
			return nil, ErrNilResource
		}
		idStr := res.ID().Value()
		if _, exists := itemMap[idStr]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateResource, idStr)
		}
		itemMap[idStr] = res
		orderList = append(orderList, idStr)
	}

	return &Resources{
		items: itemMap,
		order: orderList,
	}, nil
}

// Get returns the Resource associated with idStr.
func (rs *Resources) Get(idStr string) (*Resource, error) {
	if rs == nil {
		return nil, ErrResourceNotFound
	}
	res, ok := rs.items[idStr]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrResourceNotFound, idStr)
	}
	return res, nil
}

// Has reports whether a resource with idStr exists in the collection.
func (rs *Resources) Has(idStr string) bool {
	if rs == nil {
		return false
	}
	_, ok := rs.items[idStr]
	return ok
}

// Count returns the number of resources in the collection.
func (rs *Resources) Count() int {
	if rs == nil {
		return 0
	}
	return len(rs.items)
}

// List returns a slice of all Resource objects in deterministic insertion order.
func (rs *Resources) List() []*Resource {
	if rs == nil || len(rs.order) == 0 {
		return nil
	}
	result := make([]*Resource, len(rs.order))
	for i, key := range rs.order {
		result[i] = rs.items[key]
	}
	return result
}

// With returns a new immutable Resources collection containing the added resource.
func (rs *Resources) With(res *Resource) (*Resources, error) {
	if res == nil {
		return nil, ErrNilResource
	}

	currentList := rs.List()
	idStr := res.ID().Value()

	for _, existing := range currentList {
		if existing.ID().Value() == idStr {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateResource, idStr)
		}
	}

	newList := make([]*Resource, 0, len(currentList)+1)
	newList = append(newList, currentList...)
	newList = append(newList, res)

	return NewResources(newList...)
}
