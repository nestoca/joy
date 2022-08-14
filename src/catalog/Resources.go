package catalog

type Resources interface {
	getResourceByName(name string) *Resource
	getRequiredResourceByName(name string) (*Resource, error)
}
