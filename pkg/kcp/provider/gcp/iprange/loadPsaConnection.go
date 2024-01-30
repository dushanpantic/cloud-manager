package iprange

import (
	"context"
	"github.com/kyma-project/cloud-manager/pkg/kcp/provider/gcp/client"

	"github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	"github.com/kyma-project/cloud-manager/pkg/composed"
)

func loadPsaConnection(ctx context.Context, st composed.State) (error, context.Context) {
	state := st.(*State)
	logger := composed.LoggerFromCtx(ctx)

	ipRange := state.ObjAsIpRange()

	//If this IPRange is not for PSA, no processing is needed here.
	if ipRange.Spec.Options.Gcp != nil &&
		ipRange.Spec.Options.Gcp.Purpose != v1beta1.GcpPurposePSA {
		return nil, nil
	}

	logger.WithValues("ipRange :", ipRange.Name).Info("Loading GCP PSA Connection")

	//Get from GCP.
	gcpScope := state.Scope().Spec.Scope.Gcp
	project := gcpScope.Project
	vpc := gcpScope.VpcNetwork
	list, err := state.serviceNetworkingClient.ListServiceConnections(ctx, project, vpc)
	if err != nil {
		state.AddErrorCondition(ctx, v1beta1.ReasonGcpError, err)
		return composed.LogErrorAndReturn(err, "Error listing Service Connections from GCP", composed.StopWithRequeue, nil)
	}

	//Iterate over the list and store the address in the state object
	for _, conn := range list {
		if conn.Peering == client.PsaPeeringName {
			state.serviceConnection = conn
			break
		}
	}

	return nil, nil
}