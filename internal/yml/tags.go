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
}

var CustomTags = []string{"!lock", "!local"}

var KnownTags = append(StandardTags, CustomTags...)
