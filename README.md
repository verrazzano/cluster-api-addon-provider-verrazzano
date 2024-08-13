
# Cluster API Add-on Provider for Verrazzano
Cluster API Add-on Provider for Verrazzano extends the functionality of Cluster API by providing a solution for managing the installation, configuration, upgrade and deletion of Verrazzano on managed/workload clusters.

## Getting Started

### ⚙️ Prerequisites

Ensure there are two clusters at a minimum

- Admin cluster running the CAPI controllers.
- Workload cluster created with CAPI.


### ⚙️ Prerequisites

Refer to the development guide to build images and setup the addon provider on your cluster
- [Developer guide](DEVELOPMENT.md)


#### Install Verrazzano on Workload Cluster 

Once the addon is deployed , we can now deploy verrazzzano on the workload cluster as follows. 

Create a `VerrazzanoFleet` resource on the admin cluster. 

  ```yaml
  kubectl apply -f - <<EOF
  apiVersion: addons.cluster.x-k8s.io/v1alpha1
  kind: VerrazzanoFleet
  metadata:
    name: example-fleet-1
    namespace: default
  spec:
    clusterSelector:
      name: kluster1
    verrazzano:
      spec:
        profile: dev
    EOF
   ```
The above Resource will create a `dev` profile based Verrazzano installation on the workload cluster.

#### Remove Verrazzano from Workload Cluster

To cleanup Verrazzano installations from the remote cluster, just delete the `VerrazzanoFleet` resource. 

  ```bash
  kubectl delete verrazzanofleet example-fleet-1
  ```

## Contributing

This project welcomes contributions from the community. Before submitting a pull request, please [review our contribution guide](./CONTRIBUTING.md)

## Security

Please consult the [security guide](./SECURITY.md) for our responsible security vulnerability disclosure process

## License

*The correct copyright notice format for both documentation and software is*
    "Copyright (c) [year,] year Oracle and/or its affiliates."
*You must include the year the content was first released (on any platform) and the most recent year in which it was revised*

Copyright (c) 2023 Oracle and/or its affiliates.


Released under the Universal Permissive License v1.0 as shown at
<https://oss.oracle.com/licenses/upl/>.

