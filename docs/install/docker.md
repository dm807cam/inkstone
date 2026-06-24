Pre-built images are published to the GitHub Container Registry on every release:

```sh
docker pull ghcr.io/dm807cam/inkstone:latest
```

Available tags: `latest` (newest release), `X.Y.Z` / `X.Y` (specific versions), and `edge` (latest commit on `master`).

You can evaluate the program with:

```sh
docker run -it --rm -p 3000:3000 ghcr.io/dm807cam/inkstone:latest
```

To set it up for normal usage, you'll need to mount a volume to store user configuration and documents:

```sh
docker run -it --rm -p 3000:3000 -p 8883:8883 -v ./data:/data -e JWT_SECRET_KEY='something' ghcr.io/dm807cam/inkstone:latest
```

Explore other configuration variables on [the dedicated page](configuration.md).


## docker-compose file

```yaml
services:
  inkstone:
    image: ghcr.io/dm807cam/inkstone:latest
    container_name: inkstone
    restart: unless-stopped
    ports:
      - "3000:3000"
      - "8883:8883"
    env_file:
      - env
    volumes:
      - ./data:/data
```

For the possible environment variables, please have a look in the [configuration](configuration.md) section.


## Synology / Portainer

The image runs unmodified on Synology NAS via Container Manager or a Portainer
stack. In Portainer, create a new **Stack** and paste the compose below
(adjust the host volume path to taste):

```yaml
services:
  inkstone:
    image: ghcr.io/dm807cam/inkstone:latest
    container_name: inkstone
    restart: unless-stopped
    ports:
      - "3000:3000"
      - "8883:8883"
    environment:
      JWT_SECRET_KEY: "change-me"
      # STORAGE_URL: "https://inkstone.example.com"
    volumes:
      - /volume1/docker/inkstone:/data
```

The image is built for `linux/amd64`, which matches every Synology model that
supports Container Manager / Docker.

> If `docker pull` reports the image as not found, the GHCR package is still
> private. Open it under your GitHub profile → **Packages → inkstone → Package
> settings** and set the visibility to **Public** (one-time step).


## Rebuild the image

You can use the script `dockerbuild.sh` or there is a `make` rule:

```sh
make container
```
