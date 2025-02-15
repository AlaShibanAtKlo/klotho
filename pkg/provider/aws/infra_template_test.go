package aws

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/stretchr/testify/assert"
)

func TestInfraTemplateModification(t *testing.T) {
	cases := []struct {
		name         string
		results      []core.CloudResource
		dependencies []core.Dependency
		cfg          config.Application
		data         TemplateData
	}{
		{
			name: "simple test",
			results: []core.CloudResource{&core.Gateway{
				Name:   "gw",
				GWType: core.GatewayKind,
				Routes: []core.Route{{Path: "/"}},
			},
				&core.ExecutionUnit{
					Name:     "unit",
					ExecType: eks,
				},
			},
			cfg: config.Application{
				Provider: "aws",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"unit": {Type: eks},
				},
			},
			dependencies: []core.Dependency{},
			data: TemplateData{
				TemplateData: provider.TemplateData{
					Gateways: []provider.Gateway{
						{Name: "gw", Routes: []provider.Route{{ExecUnitName: "", Path: "/", Verb: ""}}, Targets: map[string]core.GatewayTarget(nil)},
					},
					ExecUnits: []provider.ExecUnit{
						{Name: "unit", Type: "eks", MemReqMB: 0, KeepWarm: false, Schedules: []provider.Schedule(nil), Params: config.InfraParams{}},
					},
				},
				UseVPC: true,
			},
		},
		{
			name: "helm chart test",
			results: []core.CloudResource{
				&core.ExecutionUnit{
					Name:     "unit",
					ExecType: eks,
				},
				&kubernetes.KlothoHelmChart{Values: []kubernetes.Value{{
					Type: string(kubernetes.ImageTransformation),
					Key:  kubernetes.GenerateImagePlaceholder("unit"),
				}}},
			},
			cfg: config.Application{
				Provider: "aws",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"unit": {Type: eks, HelmChartOptions: &config.HelmChartOptions{Install: true}},
				},
			},
			dependencies: []core.Dependency{},
			data: TemplateData{
				TemplateData: provider.TemplateData{
					ExecUnits: []provider.ExecUnit{
						{Name: "unit", Type: "eks", MemReqMB: 0, KeepWarm: false, Schedules: []provider.Schedule(nil),
							Params: config.InfraParams{}, HelmOptions: config.HelmChartOptions{Install: true}},
					},
				},
				UseVPC: true,
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			result := core.CompilationResult{}

			result.AddAll(tt.results)

			deps := core.Dependencies{}
			for _, dep := range tt.dependencies {
				deps.Add(dep.Source, dep.Target)
			}

			aws := AWS{
				Config: &tt.cfg,
			}

			// If we want the BuildImage to be true, we will add a dockerfile. This is because of how we initialize concurrent maps
			for _, unit := range tt.data.ExecUnits {
				res := result.Get(core.ResourceKey{Kind: core.ExecutionUnitKind, Name: unit.Name})
				resUnit, ok := res.(*core.ExecutionUnit)
				if !assert.True(ok) {
					return
				}
				resUnit.Add(&core.FileRef{FPath: "Dockerfile"})
			}

			err := aws.Transform(&result, &deps)

			if !assert.NoError(err) {
				return
			}
			awsResult := result.GetResourcesOfType(AwsTemplateDataKind)
			if !assert.Len(awsResult, 1) {
				return
			}
			data := awsResult[0]
			awsData, ok := data.(*TemplateData)
			if !assert.True(ok) {
				return
			}
			assert.Equal(tt.data.ExecUnits, awsData.ExecUnits)
			assert.Equal(tt.data.Gateways, awsData.Gateways)
			assert.Equal(tt.data.UseVPC, awsData.UseVPC)
		})
	}
}
