package catalog

import "fmt"

type Catalog struct {
	Releases []*Release
	Services []*Service
}

// Loads all resources recursively underneath given dir and returns a new catalog containing them
func NewCatalogFromDir(dir string) (*Catalog, error) {
	return nil, fmt.Errorf("not implemented")
}

// LoadDir loads all resources recursively in given dir and adds them to this catalog
func (c *Catalog) LoadDir(dir string) error {
	return fmt.Errorf("not implemented")
}

// ResolveReferences finds all objects referred by names and sets their equivalent object references for easier programmatic usage
func (c *Catalog) ResolveReferences() error {
	getResourceByName(c.Services, "")
	return fmt.Errorf("not implemented")
}

func getResourceByName(resources []*Resource, name string) {

}
