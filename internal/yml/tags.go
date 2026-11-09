package yml

var StandardTags = []string{
	"",
	"!!int",
	"!!bool",
	"!!float",
	"!!map",
	"!!str",
	"!!seq",
	"!!null",
	"!!merge",
}

const (
	TagLock             = "!lock"
	TagLocal            = "!local"
	TagOrgLocal         = "!org-local"
	TagOrgLocalPlusLock = "!org-local+lock"
	TagProtect          = "!protect"
)

var CustomTags = []string{
	TagLock,
	TagLocal,
	TagProtect,
	TagOrgLocal,
	TagOrgLocalPlusLock,
}

var KnownTags = append(StandardTags, CustomTags...)
