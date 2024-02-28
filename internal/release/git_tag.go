package release

import (
	"bytes"
	"cmp"
	"fmt"
	"text/template"

	"github.com/nestoca/joy/api/v1alpha1"
)

func GetGitTag(release *v1alpha1.Release, defaultGitTagTemplate string) (string, error) {
	gitTagTemplate := cmp.Or(release.Project.Spec.GitTagTemplate, defaultGitTagTemplate)
	if gitTagTemplate == "" {
		return release.Spec.Version, nil
	}

	tmpl, err := template.New("").Parse(gitTagTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing git tag template %q: %w", gitTagTemplate, err)
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, struct{ Release *v1alpha1.Release }{release}); err != nil {
		return "", fmt.Errorf("executing git tag template %q: %w", gitTagTemplate, err)
	}

	return buffer.String(), nil
}
