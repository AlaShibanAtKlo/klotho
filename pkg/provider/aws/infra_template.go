package aws

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/pkg/errors"
)

func (a *AWS) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	var errs multierr.Error
	data := &TemplateData{
		TemplateConfig: TemplateConfig{
			TemplateConfig: provider.TemplateConfig{
				AppName: a.Config.AppName,
			},
			PayloadsBucketName: SanitizeS3BucketName(a.Config.AppName),
		},
	}

	a.Config.UpdateForResources(result.Resources())

	data.Results = result

	helmCharts := result.GetResourcesOfType(kubernetes.HelmChartKind)

	for _, res := range result.Resources() {
		key := res.Key()
		switch res := res.(type) {
		case *core.ExecutionUnit:
			cfg := a.Config.GetExecutionUnit(key.Name)
			unit := provider.ExecUnit{
				Name:                 res.Name,
				Type:                 res.Type(),
				EnvironmentVariables: res.EnvironmentVariables,
			}

			buildImage := true
			if cfg.HelmChartOptions != nil {
				if cfg.HelmChartOptions.Install && cfg.Type != "eks" {
					errs.Append(errors.Errorf("Execution unit %s cannot be of type '%s' and helm_chart_options.install = true", unit.Name, unit.Type))
				}
				foundImageTransformation := false
				for _, c := range helmCharts {
					chart := c.(*kubernetes.KlothoHelmChart)
					for _, t := range chart.Values {
						if t.Type == string(kubernetes.ImageTransformation) && t.Key == kubernetes.GenerateImagePlaceholder(res.Name) {
							foundImageTransformation = true
							break
						}
					}
				}
				buildImage = foundImageTransformation
				unit.HelmOptions = *cfg.HelmChartOptions
			}

			if cfg.Type == "fargate" || cfg.Type == "eks" {
				data.UseVPC = true
			}

			unit.Params = cfg.InfraParams

			for _, f := range res.Files() {

				ast, ok := f.(*core.SourceFile)
				if !ok {
					continue
				}

				caps := ast.Annotations()
				for _, annot := range caps {
					cap := annot.Capability
					if cap.Name == annotation.ExecutionUnitCapability {
						if cfg.Type == "lambda" {
							unit.KeepWarm, _ = cap.Directives.Bool("keep_warm")
						}
					}
				}
			}
			if buildImage {
				data.ExecUnits = append(data.ExecUnits, unit)
			}

		case *kubernetes.KlothoHelmChart:
			data.HelmCharts = append(data.HelmCharts, provider.HelmChart{
				Directory: res.Directory,
				Values:    res.Values,
				Name:      res.Name,
			})

		case *core.StaticUnit:
			cfg := a.Config.GetStaticUnit(key.Name)
			unit := provider.StaticUnit{
				Name:          res.Name,
				Type:          res.Type(),
				IndexDocument: res.IndexDocument,
				Params:        cfg.InfraParams,
			}
			data.StaticUnits = append(data.StaticUnits, unit)

		case *core.Gateway:
			gw := provider.Gateway{
				Name:    res.Name,
				Targets: res.Targets,
			}
			for _, route := range res.Routes {
				gw.Routes = append(gw.Routes, provider.Route{
					ExecUnitName: route.ExecUnitName,
					Path:         route.Path,
					Verb:         string(route.Verb),
				})
			}
			data.Gateways = append(data.Gateways, gw)

		case *core.Persist:
			cfg := a.Config.GetPersisted(key.Name, res.Kind)

			if res.Kind == core.PersistKVKind {
				data.HasKV = true
			}
			if res.Kind == core.PersistORMKind {
				data.ORMs = append(data.ORMs, provider.ORM{
					Name:   res.Name,
					Type:   cfg.Type,
					Params: cfg.InfraParams,
				})
				if cfg.Type == "rds_postgres" {
					data.UseVPC = true
				}
			}
			if res.Kind == core.PersistRedisClusterKind || res.Kind == core.PersistRedisNodeKind {
				data.Redis = append(data.Redis, provider.Redis{
					Name:   res.Name,
					Type:   cfg.Type,
					Params: cfg.InfraParams,
				})
				data.UseVPC = true
			}

		case *core.PubSub:
			for name, event := range res.Events {
				cfg := a.Config.GetPubSub(key.Name)
				ps := provider.PubSub{
					Params:      cfg.InfraParams,
					Publishers:  event.Publishers,
					Subscribers: event.Subscribers,
					Path:        res.Path,
					VarName:     res.Name,
					EventName:   name,
				}
				data.PubSubs = append(data.PubSubs, ps)
			}

		case *core.Secrets:
			if res.Kind == core.PersistSecretKind {
				data.Secrets = append(data.Secrets, res.Secrets...)
			}
		case *core.Topology:
			data.Topology = res.GetTopologyData()
			// Make sure that these serialize to JSON as `[]` instead of `null`
			if data.Topology.EdgeData == nil {
				data.Topology.EdgeData = []core.TopologyEdgeData{}
			}
			if data.Topology.IconData == nil {
				data.Topology.IconData = []core.TopologyIconData{}
			}
		}
	}

	result.Add(data)
	return errs.ErrOrNil()
}
