package awsrediscluster

import (
	"errors"
	"strings"

	cloudcontrolv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"
	"github.com/kyma-project/cloud-manager/pkg/util"
)

func getAuthSecretName(awsRedis *cloudresourcesv1beta1.AwsRedisCluster) string {
	if awsRedis.Spec.AuthSecret != nil && len(awsRedis.Spec.AuthSecret.Name) > 0 {
		return awsRedis.Spec.AuthSecret.Name
	}

	return awsRedis.Name
}

func getAuthSecretLabels(awsRedis *cloudresourcesv1beta1.AwsRedisCluster) map[string]string {
	labelsBuilder := util.NewLabelBuilder()

	if awsRedis.Spec.AuthSecret != nil {
		for labelName, labelValue := range awsRedis.Spec.AuthSecret.Labels {
			labelsBuilder.WithCustomLabel(labelName, labelValue)
		}
	}

	labelsBuilder.WithCustomLabel(cloudresourcesv1beta1.LabelRedisClusterStatusId, awsRedis.Status.Id)
	labelsBuilder.WithCustomLabel(cloudresourcesv1beta1.LabelRedisClusterNamespace, awsRedis.Namespace)
	labelsBuilder.WithCustomLabel(cloudresourcesv1beta1.LabelCloudManaged, "true")
	labelsBuilder.WithCloudManagerDefaults()
	pvLabels := labelsBuilder.Build()

	return pvLabels
}

func getAuthSecretAnnotations(awsRedis *cloudresourcesv1beta1.AwsRedisCluster) map[string]string {
	if awsRedis.Spec.AuthSecret == nil {
		return nil
	}
	result := map[string]string{}
	for k, v := range awsRedis.Spec.AuthSecret.Annotations {
		result[k] = v
	}
	return result
}

func getAuthSecretBaseData(kcpRedis *cloudcontrolv1beta1.RedisCluster) map[string][]byte {
	result := map[string][]byte{}

	if len(kcpRedis.Status.PrimaryEndpoint) > 0 {
		result["primaryEndpoint"] = []byte(kcpRedis.Status.PrimaryEndpoint)

		splitEndpoint := strings.Split(kcpRedis.Status.PrimaryEndpoint, ":")
		if len(splitEndpoint) >= 2 {
			host := splitEndpoint[0]
			port := splitEndpoint[1]
			result["host"] = []byte(host)
			result["port"] = []byte(port)
		}
	}

	if len(kcpRedis.Status.ReadEndpoint) > 0 {
		result["readEndpoint"] = []byte(kcpRedis.Status.ReadEndpoint)

		splitReadEndpoint := strings.Split(kcpRedis.Status.ReadEndpoint, ":")
		if len(splitReadEndpoint) >= 2 {
			readHost := splitReadEndpoint[0]
			readPort := splitReadEndpoint[1]
			result["readHost"] = []byte(readHost)
			result["readPort"] = []byte(readPort)
		}
	}

	if len(kcpRedis.Status.AuthString) > 0 {
		result["authString"] = []byte(kcpRedis.Status.AuthString)
	}

	return result
}

func parseAuthSecretExtraData(extraDataTemplates map[string]string, authSecretBaseData map[string][]byte) map[string][]byte {
	baseDataStringMap := map[string]string{}
	for k, v := range authSecretBaseData {
		baseDataStringMap[k] = string(v)
	}

	return util.ParseTemplatesMapToBytesMap(extraDataTemplates, baseDataStringMap)
}

var AwsRedisClusterTierToCacheNodeTypeMap = map[cloudresourcesv1beta1.AwsRedisClusterTier]string{
	cloudresourcesv1beta1.AwsRedisClusterTierS1: "cache.t4g.small",
	cloudresourcesv1beta1.AwsRedisClusterTierS2: "cache.t4g.medium",
	cloudresourcesv1beta1.AwsRedisClusterTierS3: "cache.m7g.large",
	cloudresourcesv1beta1.AwsRedisClusterTierS4: "cache.m7g.xlarge",
	cloudresourcesv1beta1.AwsRedisClusterTierS5: "cache.m7g.2xlarge",
	cloudresourcesv1beta1.AwsRedisClusterTierS6: "cache.m7g.4xlarge",
	cloudresourcesv1beta1.AwsRedisClusterTierS7: "cache.m7g.8xlarge",
	cloudresourcesv1beta1.AwsRedisClusterTierS8: "cache.m7g.16xlarge",

	cloudresourcesv1beta1.AwsRedisClusterTierP1: "cache.m7g.large",
	cloudresourcesv1beta1.AwsRedisClusterTierP2: "cache.m7g.xlarge",
	cloudresourcesv1beta1.AwsRedisClusterTierP3: "cache.m7g.2xlarge",
	cloudresourcesv1beta1.AwsRedisClusterTierP4: "cache.m7g.4xlarge",
	cloudresourcesv1beta1.AwsRedisClusterTierP5: "cache.m7g.8xlarge",
	cloudresourcesv1beta1.AwsRedisClusterTierP6: "cache.m7g.16xlarge",
}

func redisTierToCacheNodeTypeConvertor(AwsRedisClusterTier cloudresourcesv1beta1.AwsRedisClusterTier) (string, error) {
	cacheNode, exists := AwsRedisClusterTierToCacheNodeTypeMap[AwsRedisClusterTier]

	if !exists {
		return "", errors.New("unknown redis tier")
	}

	return cacheNode, nil
}
