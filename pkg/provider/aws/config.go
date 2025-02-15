package aws

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
)

type (
	TemplateConfig struct {
		provider.TemplateConfig
		PayloadsBucketName string
	}

	TemplateData struct {
		provider.TemplateData
		TemplateConfig
		UseVPC bool
	}
)

var AwsTemplateDataKind = "aws_template_data"

func (*TemplateData) Type() string { return "" }

func (t *TemplateData) Key() core.ResourceKey {
	return core.ResourceKey{
		Name: t.AppName,
		Kind: AwsTemplateDataKind,
	}
}

func (c *AWS) Name() string { return "aws" }

// Enums for the types we allow in the aws provider so that we can reuse the same string within the provider
const (
	eks                    = "eks"
	fargate                = "fargate"
	lambda                 = "lambda"
	apigateway             = "apigateway"
	rds_postgres           = "rds_postgres"
	s3                     = "s3"
	dynamodb               = "dynamodb"
	elasticache            = "elasticache"
	memorydb               = "memorydb"
	sns                    = "sns"
	cockroachdb_serverless = "cockroachdb_serverless"
)

var defaultConfig = config.Defaults{
	ExecutionUnit: config.KindDefaults{
		Type: lambda,
		InfraParamsByType: map[string]config.InfraParams{
			lambda: {
				"memorySize": 512,
				"timeout":    180,
			},
			fargate: {
				"memory": 512,
				"cpu":    256,
			},
			eks: {
				"nodeType": "fargate",
				"replicas": 2,
			},
		},
	},
	StaticUnit: config.KindDefaults{
		Type: s3,
		InfraParamsByType: map[string]config.InfraParams{
			s3: {
				"cloudFrontEnabled": true,
			},
		},
	},
	Expose: config.KindDefaults{
		Type: apigateway,
	},
	PubSub: config.KindDefaults{
		Type: sns,
	},
	Persist: config.PersistKindDefaults{
		KV: config.KindDefaults{
			Type: dynamodb,
		},
		FS: config.KindDefaults{
			Type: s3,
		},
		Secret: config.KindDefaults{
			Type: s3,
		},
		ORM: config.KindDefaults{
			Type: rds_postgres,
			InfraParamsByType: map[string]config.InfraParams{
				rds_postgres: {
					"instanceClass":     "db.t4g.micro",
					"allocatedStorage":  20,
					"skipFinalSnapshot": true,
					"engineVersion":     "13.7",
				},
				cockroachdb_serverless: {},
			},
		},
		RedisNode: config.KindDefaults{
			Type: elasticache,
			InfraParamsByType: map[string]config.InfraParams{
				elasticache: {
					"nodeType":      "cache.t3.micro",
					"numCacheNodes": 1,
				},
			},
		},
		RedisCluster: config.KindDefaults{
			Type: memorydb,
			InfraParamsByType: map[string]config.InfraParams{
				memorydb: {
					"nodeType":            "db.t4g.small",
					"numReplicasPerShard": "1",
					"numShards":           "2",
				},
			},
		},
	},
}

func (a *AWS) GetDefaultConfig() config.Defaults {
	return defaultConfig
}

// GetKindTypeMappings returns a list of valid types for the aws provider based on the kind passed in
func (a *AWS) GetKindTypeMappings(kind string) ([]string, bool) {
	switch kind {
	case core.ExecutionUnitKind:
		return []string{eks, fargate, lambda}, true
	case core.GatewayKind:
		return []string{apigateway}, true
	case core.StaticUnitKind:
		return []string{s3}, true
	case string(core.PersistFileKind):
		return []string{s3}, true
	case string(core.PersistKVKind):
		return []string{dynamodb}, true
	case string(core.PersistORMKind):
		return []string{rds_postgres}, true
	case string(core.PersistRedisNodeKind):
		return []string{elasticache}, true
	case string(core.PersistRedisClusterKind):
		return []string{memorydb}, true
	case string(core.PersistSecretKind):
		return []string{s3}, true
	case core.PubSubKind:
		return []string{sns}, true
	}
	return nil, false
}
