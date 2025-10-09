
The project provides K8s Custom Resources, and their controllers.

Control plane resources can be found in the api/cloud-control dir.
Runtime (agent) plane resources can be found in the api/cloud-resources dir.

Controller implementations for control plane are inside pkg/kcp dir.
Controller implementations for runtime plane are inside pkg/skr dir.

Project should be extended with support for Azure Managed Redis.

There is already Azure Cache for Redis implemented, and we can use that implementation for steering.

Implementation can be found on following places:
1. Client implementation is located in pkg/kcp/provider/azure/client/clientRedis.go file
2. Reconciler for control plane is located on pkg/kcp/provider/azure/redisinstance path.
3. Reconciler for runtime plane is located on pkg/skr/azureredisinstance

