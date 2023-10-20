//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE
package promote

import "github.com/nestoca/joy/internal/yml"

type YamlWriter interface {
	Write(file *yml.File) error
}
