package references

import (
	"errors"
	"fmt"

	"github.com/nestoca/joy/internal/style"
)

type MissingError struct {
	SourceKind      string
	SourceName      string
	DestinationKind string
	DestinationName string
}

func NewMissingError(sourceKind, sourceName, destinationKind, destinationName string) *MissingError {
	return &MissingError{
		SourceKind:      sourceKind,
		SourceName:      sourceName,
		DestinationKind: destinationKind,
		DestinationName: destinationName,
	}
}

func (e *MissingError) Error() string {
	return fmt.Sprintf("%s %s is referencing missing %s %s", e.SourceName, e.SourceKind, e.DestinationName, e.DestinationKind)
}

func (e *MissingError) StyledError() string {
	return fmt.Sprintf("%s %s is referencing missing %s %s",
		style.ResourceKind(e.SourceKind),
		style.Resource(e.SourceName),
		style.ResourceKind(e.DestinationKind),
		style.Resource(e.DestinationName))
}

func AsMissingErrors(err error) []*MissingError {
	var errs []*MissingError
	for {
		if err == nil {
			break
		}

		var missingErr *MissingError
		if errors.As(err, &missingErr) {
			errs = append(errs, missingErr)
		}

		// Unwrap to get the next error in the chain.
		err = errors.Unwrap(err)
	}
	return errs
}
