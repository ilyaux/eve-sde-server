# Releasing

Releases are tag-driven. A release tag builds cross-platform archives, publishes
a multi-architecture Docker image to GHCR, and creates a GitHub release with
checksums.

## Release Checklist

1. Make sure `main` is green in CI.
2. Choose a semantic version tag such as `v1.0.0`.
3. Create and push the tag:

   ```bash
   git switch main
   git pull --ff-only
   git tag v1.0.0
   git push origin v1.0.0
   ```

4. Wait for the `Release` workflow to finish.
5. Verify the GitHub release contains archives and `checksums.txt`.
6. Verify the Docker image exists at `ghcr.io/ilyaux/eve-sde-server:v1.0.0`.

The workflow can also be run manually with `workflow_dispatch`, but the
requested version must already exist as a tag.

## Local Artifact Build

To build the same release archives locally:

```bash
VERSION=v1.0.0 make release-assets
```

The script writes archives and `checksums.txt` to `dist/`. It builds:

- `eve-sde-server`
- `eve-sde-migrate`
- `eve-sde-import-sde`

Targets:

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`

Local builds require Bash, Go 1.25+, `tar`, `zip`, and `sha256sum`.

## Docker Image

Release images are published to GitHub Container Registry:

```bash
docker pull ghcr.io/ilyaux/eve-sde-server:v1.0.0
docker run --rm -p 8080:8080 ghcr.io/ilyaux/eve-sde-server:v1.0.0
```

The image includes the server, migration binary, import binary, OpenAPI spec,
and admin/static web files. On startup it runs migrations before starting the
server.
