## ⚙️  Building the Project
Intended audience: developers who want to build images locally and try out the Verrazzano addon provider in a local cluster.


```shell
git clone https://github.com/verrazzano/cluster-api-adddon-provider-verrazzano.git && cd $_
export TAG="<image tag>"
export REGISTRY="<image-registry>" # default is ghcr.io/verrazzano
make verrazzano-addon-build
```

These commands will create the release manifests and also push the images for the Verrazzano addon provider to the specified registry and tag.

The release artifacts are created in the following folder structure:

```shell
release/
├── addon-verrazzano
│   └── v1.0.0
│       ├── addon-components.yaml
│       └── metadata.yaml
```

The above folder structure can then be referenced in the `clusterctl` configuration file `~/.cluster-api/clusterctl.yaml`.

```shell
providers:
  - name: "verrazzano"
    type: "AddonProvider"
    url: "${GOPATH}/src/github.com/verrazzano/cluster-api-addon-provider-verrazzano/release/addon-verrazzano/v1.0.0/addon-components.yaml"
```

Then initialize `clusterctl` with the locally built Verrazzano addon provider:

```
clusterctl init --bootstrap ocne --control-plane ocne --addon verrazzano -i oci:v0.13.0
```