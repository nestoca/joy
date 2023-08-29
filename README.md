# joy

Joy is a GitOps CLI tool with CRD-style YAML file formats for simplifying the deployment of project releases to Kubernetes and their promotion across environments. It proves a perfect complement to other GitOps tools, such as [ArgoCD](https://argoproj.github.io/cd/) and [Crossplane](https://www.crossplane.io/).

- Defines CRD-style YAML file formats:
  - `Environment`: A Kubernetes cluster and/or namespace to deploy to.
  - `Project`: An application (eg: a microservice, front-end...) that can be deployed to different environments.
  - `Release`: The deployment configuration for a specific version of a project to a given environment.
- GitOps-oriented: Your joy "catalog" git repo defines the Source of Truth™ for your environments, projects and releases. Changes to releases can be detected by a CD tool such as ArgoCD, transformed and deployed to the target environment.
- The `joy` CLI allows you to:
  - List and select environments, projects, releases.
  - Promote releases from one environment to another, transfering specific portions of release configuration intelligently.
  - Query who owns given projects/releases, optionally integrating with [jac](https://github.com/nestoca/jac) for rich people metadata.
- The [joy-generator](https://github.com/nestoca/joy-generator) ArgoCD `ApplicationSet` generator plugin allows to automatically generate ArgoCD `Application` resources from your joy catalog, to deploy your releases to Kubernetes.
- YAML resources are extensible via custom values to address your specific metadata needs.

# Why joy?

Joy dramatically reduces the cognitive load on developers when it comes to deploying their projects to Kubernetes. It allows them to think in terms of high-level release YAML files, and then to use the `joy` CLI to promote those releases across environments.

It also provides the DevOps/platform engineers with a layer of abstraction to manage the more intricate deployment details, such as ArgoCD Applications and Helm charts.

What makes joy releases so useful and powerful:
- Because they only contain the high-level information that is relevant to developers, they provide a very simple, visual and intuitive interface to the deployment process.
- Portions of the release that are environment-specific can be locked with a special `# lock` comment to prevent them from being carried over from environment to environment during promotions. 
- They tie together all the deployment configuration that needs to travel with a given build, such as:
  - Container image+version
  - Helm chart+version
  - Helm values for deploying the container
  - And potentially values for provisioning related-infrastructure (eg: via Crossplane)

# How does it work?

- DevOps/platform engineers create a "joy catalog" git repo, defining different `Environment` resources.
- Developers create `Project` and `Release` resources in that same git repo.
- `Release` resources placed within same sub-directory tree as an `Environment` resource are considered part of that environment.
- A `Release` defines:
  - The release's name.
  - The project it belongs to.
  - The version of the container to deploy.
  - The Helm chart to use for deployment.
  - The values to pass to the Helm chart.
  - Certain environment-specific portions of the release can be marked with a `# lock` comment to exclude them from promotions.
- CI pipelines for `master` branch build call `joy build promote` at the end of their process to promote the release in

# Using joy with ArgoCD

Joy was designed with [ArgoCD]() in mind, even if it does not depend on it and could be used with other CD tools, such as FluxCD.

# Using joy with Crossplane

Integrating a tool like [Crossplane](https://www.crossplane.io/) with joy allows you to provision the infrastructure required by your projects as part of the same release process as its deployment. This is a powerful way to ensure that your infrastructure is always in sync with your project deployments.

Take the example of an project that's just been modified to now require a storage bucket. Whenever that new version will be deployed to a new environment, it will require that bucket to be provisioned. With joy, that infrastructure requirement can be defined in the `Release` resource values and that bucket can be provisioned automatically whenever that new version of the project gets promoted to a new environment. That is a game changer, as it can be extremely tricky to try and keep track of the evolving infrastructure requirements of your projects as they get promoted across environments.

The easiest way to achieve that is to rely on the helm chart to create the appropriate Crossplane resources required by the project.

# Using joy with jac

# Using joy with sealed-secrets