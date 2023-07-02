package release

type Filter interface {
	Match(release *Release) bool
}
