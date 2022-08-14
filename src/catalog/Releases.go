package catalog

type Releases []*Release

func (r Releases) getResourceByName(name string) (*Resource, error) {
	for release := range r {
		if release.
	}
}
