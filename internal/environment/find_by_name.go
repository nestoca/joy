package environment

import "github.com/nestoca/joy/api/v1alpha1"

func FindByName(environments []*v1alpha1.Environment, name string) *v1alpha1.Environment {
	for _, env := range environments {
		if env.Name == name {
			return env
		}
	}
	return nil
}
