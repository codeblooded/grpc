# Container Image Names

The benchmarking framework uses docker containers for each of the test
components. Since images are built and pushed to a docker registry often, the
number of images can grow large. Therefore, this spec defines a naming
convention that can aid automated clean-up scripts.

## Background

Docker images are encapsulated in repositories in a registry. The naming is a
bit confusing. In terms of the Docker CLI, a tag is `"httpd:v1"` where `"v1"`
is the version. This differs from the Registry perspective, where `"httpd"` is
a repository and `"v1"` is a tag.

Docker registries, like Google Container Registry ([gcr.io]) and DockerHub
([hub.docker.com]), implement a common [Registry API]. However, docker and this
API do not natively provide batch clean-up actions. Docker also provides no TTL
on container images.

[gcr.io]: https://gcr.io
[hub.docker.com]: https://hub.docker.com
[Registry API]: https://docs.docker.com/registry/spec/api/

One approach for building a TTL on container images would be to use timestamps
as versions/tags of the images. However, the Registry API cannot query based on
these. Checking these requires a separate API call for each repository.

Instead of excessive requests, this framework imposes a strict set of naming
conventions that can aid in automated clean up.

## Specification

The framework allows a repository to contain exactly 1 container image. The
repository is named as follows:

```
[ REGISTRY_PREFIX "/" ]
    "benchmarks" "/"
          <USER> "/"
              <TIMESTAMP> "/"
                  <COMPONENT_KIND> "/"
                      <LANGUAGE_CODE> [:latest]
```

1.  The name begins with an optional `REGISTRY_PREFIX`. Some registries, like
    Google Container Registry (GCR) require a specific prefix. This allows
    images to be uploaded to these registries.

2.  The word `"benchmarks"` helps organize images. Projects may have many
    images, and this keyword shows which are related to performance tests. In
    GCR, these repositories are displayed in a nested form, like a subdirectory.

3.  Each container contains a component's executable that was built after
    cloning a GitHub repository. The `USER` specifies the username or
    organization of this repository. For example, the grpc/grpc-java repository
    would set the user to "grpc". A developer's fork, like codeblooded/grpc,
    would set the user to "codeblooded".

4.  The `TIMESTAMP` is the time when the set of images were built, represented in
    seconds since the unix epoch.

5.  The `COMPONENT_KIND` specifies what the image contains. For example, this
    may be `"driver"` for the test orchestrator and `"worker"` for a client or
    server.

6.  The `LANGUAGE_CODE` indicates the programming language of the component.
    This differentiates two components that may be different but built
    simultaneously. It is omitted for component types which only have 1
    language, like the `"driver"`. An example would be a load test which uses
    both a worker in Go and a worker in C++.

7.  The only relevant version/tag is `"latest"`, because only one image lives in
    a repository.

The user and component kind allow developers to quickly see who built the image
and what the image contains. This also provides a way for automated scripts to
set different TTLs based on the component kind or user. For example, the images
of a continuous integration bot may have different lifetimes than those of a
developer.

Including the timestamp greatly reduces the network requests (and, therefore,
complexity) of the clean up tool. Both approaches require fetching a list of all
repositories and deletions of relevant images. They differ in the number of
requests for determining deletion eligibility. One approach, using versions/tags
to specify a timestamp, requires a request for every single repository. This
includes uneligible repositories. Including the timestamp in the name, the
approach adopted by this spec, removes all of these eligibility requests. The
name alone determines eligibility. So, the preliminary list of all repositories
is sufficient. As an additional bonus, it allows developers to quickly see what
components were built together.

### Other Considerations

Repository names can grow fairly long, and Docker used to cap names at 30
characters. However, this cap has been increased to 255. See [moby/moby#10392].

[moby/moby#10392]: https://github.com/moby/moby/issues/10392

To keep names minimal, the number of seconds since the unix epoch is used as the
timestamp. This will remain just 10 characters until the year 2286. Usernames
are generally capped at a few dozen characters, and languages use their shortest
recognizable form:

| Official Name        | Language Code |
| :------------------: | :-----------: |
| C++                  | cc            |
| C# (C core)          | cs            |
| C# (.NET)            | dotnet        |
| Dart                 | dart          |
| Go                   | go            |
| Java                 | java          |
| JavaScript (web)     | js            |
| JavaScript (Node.js) | node          |
| Kotlin               | kt            |
| Objective-C          | objc          |
| PHP                  | php           |
| Python               | py            |
| Ruby                 | rb            |
| Swift                | swift         |
