# Joy

Joy is a GitOps CLI tool with Kubernetes-style YAML file formats for simplifying the deployment of project releases to Kubernetes and their promotion across environments. It proves a perfect complement to other GitOps tools, such as [ArgoCD](https://argoproj.github.io/cd/) and [Crossplane](https://www.crossplane.io/).

- Defines Kubernetes-style YAML file formats:
  - `Environment`: A Kubernetes cluster and/or namespace to deploy to.
  - `Project`: An application (eg: a microservice, front-end...) that can be deployed to different environments.
  - `Release`: The deployment configuration for a specific version of a project to a given environment.
- GitOps-oriented: Your joy "catalog" git repo defines the Source of Truth™ for your environments, projects and releases. Combined with the ApplicationSet generator plugin, changes to releases are detected by ArgoCD, transformed and deployed to the target environment.
- The `joy` CLI allows you to:
  - List and select environments, projects, releases.
  - Promote releases from one environment to another, transfering specific portions of release configuration intelligently.
  - Query who owns given projects/releases, optionally integrating with [jac](https://github.com/nestoca/jac) for rich people metadata.
- The [joy-generator](https://github.com/nestoca/joy-generator) ArgoCD `ApplicationSet` generator plugin allows to automatically generate ArgoCD `Application` resources from your joy catalog, to deploy your releases to Kubernetes.
- YAML resources are extensible via custom values to address your specific metadata needs.

# Why Joy?

Joy dramatically reduces the cognitive load on developers when it comes to deploying their projects to Kubernetes. It allows them to think in terms of high-level release YAML files, and then to use the `joy` CLI to promote those releases across environments.

It also provides the DevOps/platform engineers with a layer of abstraction to manage the more intricate deployment details, such as ArgoCD Applications and Helm charts.

What makes joy releases useful and powerful:

- They only contain the high-level information that is relevant to developers, providing a very simple, visual and intuitive interface to the deployment process.
- Portions of the release that are environment-specific can be locked with the `!lock` custom yaml tag to prevent them from being carried over from environment to environment during promotions.
- They tie together all the deployment configuration that needs to travel with a given build, such as:
  - Container image+version
  - Helm chart+version
  - Helm values for deploying the container

# Installing Joy

## Installing with homebrew

```bash
$ brew tap nestoca/public
$ brew install joy
$ joy setup
```

Upgrade with:

```
 brew upgrade joy
```

## Installing manually

Download from GitHub [releases](https://github.com/nestoca/joy/releases/latest), put the binary somewhere in your `$PATH` and then run:

```
$ joy setup
```

# How does it work?

- DevOps/platform engineers create a Joy "catalog" git repo, defining different `Environment` resources.
- Developers create `Project` and `Release` resources in that same git repo.
- `Release` resources placed within same sub-directory tree as an `Environment` resource are considered part of that environment.
- A `Release` describes the high-level deployment configuration of a given project in a specific environment, including container image version, helm chart and values.
- Certain environment-specific values of a `Release` can be marked with a `!lock` custom yaml tag to exclude them from cross-promotions.
- CI pipelines call `joy build promote` at the end of their build process to automatically promote new version of project in its corresponding release(s) of some target environment.
- Developers call `joy release promote` to cross-promote their releases from one environment to another.
- ArgoCD watches catalog repo, fetches changes to joy resources and transforms them into Argo `Application` resources (via the [joy-generator](https://github.com/nestoca/joy-generator) plugin), which get deployed into different environments.

# Catalog

The catalog is a git repository where you define your environments, your projects and their releases. ArgoCD watches that repository and takes care of propagating changes to corresponding environments.

## Environments

Joy `Environments` are CRD-like yaml resources defining target deployment environments for your `Projects` / `Releases`.

They are of the form:

```yaml
apiVersion: joy.nesto.ca/v1alpha1
kind: Environment
metadata:
  name: dev
spec:
  order: 2
  promotion:
    fromPullRequests: true
    fromEnvironments:
      - staging
      - demo
  owners:
    - stream-origination
    - stream-servicing
```

* `name`: technical name of the environment.

* `order`: used to sort environments when displaying them in columns or lists.

* `promotion`: tells the `joy` cli which sources are allowed to have their releases promoted to this environment.
  
  * `fromPullRequests`: boolean indicating whether pull requests can be promoted to this environment (via `joy pr promote` command).
  
  * `fromEnvironments` list of environments that can be promoted to this environment (via `joy release promote` command).

* `owners`: list of streams, teams or specific people owning this environment. It can be any identifiers you want, but if you want to integrate with [Jac](https://github.com/nestoca/jac), these should be the names of jac groups (see: [Using joy with jac](#using-joy-with-jac)).

> All releases in same directory as an environment resource file, or any sub-directory, are automatically considered part of that environment.

## Projects

Joy `Projects` are CRD-like yaml resources that correspond to something that can be built and deployed. They conceptually group together multiple `Releases` of the same thing being deployed to multiple `Environments`.

They are of the form:

```yaml
apiVersion: joy.nesto.ca/v1alpha1
kind: Project
metadata:
  name: podinfo
spec:
  owners:
    - stream-devops
```

For example, the last step of CI should be a call to `joy build promote`, passing it the project name, and the target environment. Joy then resolves all releases associated with that project in given environment, and promotes those releases to latest version of image just built.

## Releases

Joy `Releases` are CRD-like yaml resources defining the deployment configuration of a given `Project` in a given `Environment`.

They are of the form:

```yaml
apiVersion: joy.nesto.ca/v1alpha1
kind: Release
metadata:
  name: podinfo
spec:
  version: '6.5.2'
  versionKey: 'image.tag'
  chart:
    name: 'podinfo'
    repoUrl: 'https://stefanprodan.github.io/podinfo'
    version: '6.5.2'
  values:
    replicaCount: 2
    backend: http://backend-podinfo:9898/echo
```

### Defining multiple releases of same project

Most of the time, each project will have a single release of itself in any given environment, with same name (eg: `podinfo` project deployed to a `podinfo` release in each environment).

However, you may sometimes need _multiple instances_ of some project to be deployed within each environment, each with its own configurations.

Imagine a `backup` project for a container that you want to deploy as different cronjobs, with different schedules and target buckets. In that case, we could define two releases, `backup-daily` and `backup-monthly`, both referencing the same `backup` project, but configured differently:

`backup-daily.yaml`:

```yaml
apiVersion: joy.nesto.ca/v1alpha1
kind: Release
metadata:
  name: backup-daily
spec:
  project: backup
  version: '0.8.0'
  chart:
    name: backup-cronjob
    repoUrl: https://my-chart-repo
    version: '1.0.1'
  values:
    schedule: @daily
    bucket: /backups/daily
```

`backup-monthly.yaml`:

```yaml
apiVersion: joy.nesto.ca/v1alpha1
kind: Release
metadata:
  name: backup-monthly
spec:
  project: backup
  version: '0.8.0'
  chart:
    name: backup-cronjob
    repoUrl: https://my-chart-repo
    version: '1.0.1'
  values:
    schedule: @monthly
    bucket: /backups/monthly
```

# Setting up CI

Your CI pipeline for a hypothetical project (eg: `PROJECT=acme`) could have this general structure:

1. Test, build and publish your artifact to some registry, using an incrementing version number (eg: `VERSION=0.123.0`).

2. Call `joy build promote` to auto-promote this new build version to some environment. For main branch, that could be your pre-prod environment (eg: `ENV=staging`) and for a PR branch, that could be some dev environment (eg: `ENV=dev`):
   
   ```bash
   $ joy build promote $ENV $PROJECT $VERSION
   ```

For PR branches, we recommend using GitHub labels of the form `promote:<ENV>` (eg: `deploy:dev`) on PRs to determine the target environment. Joy provides the `joy pr promote` command to manage those labels:

```
$ joy pr promote
? No pull request found for current branch, create one? Yes
Creating pull request for my-branch into master in acme/my-project

? Title Update README.md
? Body <Received>
? What's next? Submit
https://github.com/acme/my-project/pull/13
? Select environment to auto-promote builds of pull request to:  [Use arrows to move, type to filter]
  [none]
> dev
  staging
```

This will create a PR for current branch and add the `promote:dev` label to it.

Note that it's up to you to parse the PR labels in your CI pipeline and determine the appropriate promotion environment.  Joy merely provides the `joy pr promote` command to simplify managing the `promote:<ENV>` labels on your GitHub PRs.

# # ...

# Setting up CD

# Troubleshooting issues

Joy provides the `joy diagnose` command to troubleshoot your joy setup:

```
$ joy diagnose
✅ Executable
  ➡️ Version: v0.19.1
  ✅ Version meets minimum of v0.18.1 required by catalog
  ➡️ File path: /opt/homebrew/bin/joy

✅ Dependencies
  Required dependencies
    ✅ gh installed
    ✅ git installed

  Optional dependencies
    ✅ jac installed
    ✅ kubectl installed

✅ Config
  ✅ File exists: /Users/john/.joyrc
  ➡️ Selected environments: 3
  ➡️ Selected releases: <all>

✅ Catalog
  Git working copy
    ➡️ Directory exists: /Users/john/dev/catalog
    ✅ Working copy is valid
    ✅ Working copy has no uncommitted changes
    ✅ Default branch master is checked out
    ✅ Default branch is in sync with remote
    ➡️ Current commit: 1b8babf9fcddfad19928679f0b26921674f349ab

  Loading catalog
    ✅ Catalog detected
    ✅ Catalog loaded successfully

  Resources
    ➡️ Environments: 6
    ➡️ Projects: 20
    ➡️ Releases: 25

  Cross-references
    ✅ All resource cross-references resolved successfully

🚀 All systems nominal. Houston, we're cleared for launch!
```

# Using joy with jac

[Jac](https://github.com/nestoca/jac) is a GitOps CLI tool and YAML file format for managing and querying your people metadata Source of Truth™.

In order to use jac with joy, simply configure your joy projects with owners that correspond to jac group identifiers, for example:

```yaml
apiVersion: joy.nesto.ca/v1alpha1
kind: Project
metadata:
  name: podinfo
spec:
  owners:
    - team-dragons
```

You can then use `joy project owners` and `joy release owners` to have jac resolve the actual people owning those projects and releases:

```bash
$ joy release owners
Select release:
  service-1
  service-2
> podinfo-1
  podinfo-2

  NAME          FIRST NAME  LAST NAME  EMAIL              GROUPS                      INHERITED GROUPS
  jack-sparrow  Jack        Sparrow    jack@example.com   DevOps Dragons              Tech Support
  peter-pan     Peter       Pan        peter@example.com  Backend Developer Dragons   Tech Support
 ———
 Count: 2
```

(shorthand: `joy proj own` and `joy rel own`)

Arbitrary arguments and flags following those two commands will be passed directly to jac, allowing you to leverage all of jac's querying and filtering capabilities.

Note that the `jac` CLI must be installed and configured on your machine in order for this to work.

# Using joy with Sealed Secrets

[Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets) is an open-source Kubernetes controller and client-side CLI tool from Bitnami that aims to solve the problem of storing secrets in Git using asymmetric crypto encryption. It generates a public and private key and can be used to encrypt/decrypt application secrets in a secure way. Sealed Secrets are "one-way" encrypted K8s Secrets that can be created by anyone but can only be decrypted by the controller running in the target cluster recovering the original object.

## Importing sealed secrets certificates into your joy catalog

Joy takes advantage of the fact that the public key (or "certificate") for each cluster — required by developers to encrypt secrets — is not sensitive and can safely be stored in your joy git catalog in each `Environment` resource, for example:

```yaml
apiVersion: joy.nesto.ca/v1alpha1
kind: Environment
metadata:
  name: staging
spec:
  order: 0
  owners:
    - team-dragons
  sealedSecretsCert: |
    -----BEGIN CERTIFICATE-----
    MIIEaDCCZrSgZwIBZgIQQVigtDuTUYj4m2BxtpZbPaZNBgkqhkiG9w0BZQsFZDZZ
    MB4XDTIyMTZwNaIwNTZyN1oXDTMyMTZwNDIwNTZyN1owZDCCZiIwDQYJKoZIhvcN
...
    kwqbLB/f7nqjUZvN6fkjqKCleLx4GI3H8WZxURqYU2ZPQ6/6kmNZHjekiYRJYhdn
    QpgDdtXavNV/KcrL1je/BZErRusBbjR5LwFaZp0YCo0=
    -----END CERTIFICATE-----
```

Joy provides a command to help you automatically extract your cluster's public key and add it to one of your `Environment` resources, assuming you have proper permissions to read secrets from given cluster:

```bash
$ joy sealed-secrets import

? Select kube context of cluster to fetch seal secrets certificate from:
  dev-rw
  dev-admin
  staging-rw
> staging-admin
  production-rw
  production-admin

? Select environment to import sealed secrets certificate into:
  dev
> staging
  production

✅ Imported sealed secrets certificate from cluster staging-admin into environment staging
Make sure to commit and push those changes to git.
```

## Encrypting secrets

Once the sealed secrets certificate has been imported and committed into your joy catalog, developers can use the `joy sealed-secrets seal` command to encrypt their secrets for a specific environment:

```bash
$ joy sealed-secrets seal

? Select environment to seal secret in:
  dev
> staging
  production

? Enter secret to seal [Enter 2 empty lines to finish]
This is my highly
sensitive secret
which comprises
multiple lines


🔒 Sealed secret:
AgAWCq7OKWq+eabiqCwccMr0ll9e07Rwc1iW7itKyc1AzQlLmjrr1UK/VHP8ALWAWODSrJ8E8WVs4qvTjUdLVpbLRZVVBajYUAUXCxeVtk
QciuWsl5tfuWl4yxZdeBuTIsy/DHdK37STmRFB7yJeqqozPG+VyiJEb/to+jEpp7hzTkyk1GrPOz3Rtw4RUcwHD1lGtBdlanIDTd6B76Ju
...
+WXmUf3cUW8tSF5nMGY8f9hv4Wmb4cCa1yCC1QaVv82qlWSfKGsxo2oVMFvxwPXpqXUzfVY/TF01XrWoj4uwMDbvXlXNvooS7KpHcXpS6s
bAb8BXekgmMPhnGMH3OmCHPREcKEN4ccc+gDbOFp7mQjnnxYdMAmxQKY42zN
```

Note that the `kubeseal` CLI must be installed on your machine in order for this to work.

# Combining deployments and infrastructure provisioning

Integrating a tool like [Crossplane](https://www.crossplane.io/) with joy allows you to provision the infrastructure required by your projects as part of the same release process as their deployment. This is a powerful way to ensure that your infrastructure is always in sync with your project deployments.

Take the example of a project that's just been modified to now require a storage bucket. Whenever that new version will be deployed to a new environment, it will require that bucket to be provisioned in that environment. With joy, that infrastructure requirement can be defined in the `Release` resource values and that bucket can be provisioned automatically whenever that new version of the project gets promoted to a new environment. That is a game changer, as it can be extremely tricky to try and keep track of the evolving infrastructure requirements of your projects as they get promoted across environments.

As Crossplane is not yet very flexible in terms of templating (eg: conditionals and loops), we recommend relying on helm charts to deal with this templating complexity and create the appropriate Crossplane resources required by the project based on the values passed to it. For example, given a `bucketName` value, the helm chart could create a Crossplane `Bucket` resource and let the Crossplane provider provision the actual bucket in the cloud.
