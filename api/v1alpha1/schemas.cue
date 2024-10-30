package v1alpha1

#release: {
	apiVersion: #apiVersion
	kind:       "Release"
	metadata:   #metadata
	spec!: {
		version: string
		project: string

		// Chart to be used by release. If omitted will use the default chart reference defined by the catalog.
		// Two forms are allowed, you can select a chart reference from the catalog and optionally override its version. Otherwise a fully qualified chart must be defined.
		chart?: ({
			ref?:     string
			repoUrl?: string
			name?:    string
			version?: string
		})

		// Namespace to deploy the release to. If omitted will use the default namespace defined in application set template.
		namespace?: string

		// values are the values used with the chart of this release.
		// To see the the spec for the values associated to a chart for a given release run:
		// joy rel schema --env $ENV $RELEASE
		values: [string]: _

		// links is the map of release-level overrides and additions for release links defined in project and/or catalog configuration.
		links?: [string]: string
	}
}

#environment: {
	apiVersion: #apiVersion
	kind:       "Environment"
	metadata!:  #metadata
	spec: {
		order?: number
		promotion?: {
			allowAutoMerge?:   bool
			fromPullRequests?: bool
			fromEnvironments?: [...string]
		}
		cluster?:   string
		namespace?: string

		// ChartVersions allows the environment to override the given version of the catalog's chart references.
		// This allows for environments to roll out new versions of chart references.
		chartVersions?: [string]: string
		owners?: [...string]

		// SealedSecretsCert is the public certificate of the Sealed Secrets controller for this environment
		// that can be used to encrypt secrets targeted to this environment using the `joy secret seal` command.
		sealedSecretsCert?: string

		// Values are the environment-level values that can optionally be injected into releases' values during rendering
		// via the `$ref(.Environment.Spec.Values.someKey)` or `$spread(...)` template expressions.
		values?: [string]: _
	}
}

#project: {
	apiVersion: #apiVersion
	kind:       "Project"
	metadata!:  #metadata
	spec: {
		// Owners is the list of identifiers of owners of the project.
		// It can be any string that uniquely identifies the owners, such as email addresses or backstage group names
		owners?: [...string]

		// Reviewers is the list of GitHub users who should always added a reviewer for the project.
		reviewers?: [...string]
		repository?: string

		// Location of the project files in the repository. Should be empty if the whole repository is the project.
		// If there is more than one location, specify the main subdirectory of the project first.
		repositorySubpaths?: [...string]

		// GitTagTemplate allows you to configure what your git tag look like relative to a release via go templates
		// example: gitTagTemplate: api/v{{ .Release.Spec.Version }}
		gitTagTemplate?: string

		// Links is the map of project-level overrides and additions for project links defined in catalog configuration.
		links?: [string]: string

		// ReleaseLinks is the map of project-level overrides and additions for release links defined in catalog configuration.
		releaseLinks?: [string]: string
	}
}

#apiVersion: "joy.nesto.ca/v1alpha1"

#metadata: {
	name!: string
	annotations?: [string]: string
	labels?: [string]:      string
}
