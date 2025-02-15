import * as pulumi_k8s from '@pulumi/kubernetes'
import * as pulumi from '@pulumi/pulumi'
import { CloudCCLib } from '../../deploylib'
import { Eks } from '../eks'

export class Value {
    public ExecUnitName: string
    public Kind: string
    public Type: string
    public Key: string
}

enum ValueTypes {
    TargetGroupTransformation = 'target_group',
    ImageTransformation = 'image',
    EnvironmentVariableTransformation = 'env_var',
    ServiceAccountAnnotationTransformation = 'service_account_annotation',
}

export const getChartValues = (
    lib: CloudCCLib,
    eks: Eks,
    transformations: Value[]
): {
    [x: string]: any
} => {
    const values = {}
    transformations.forEach((t: Value) => {
        switch (t.Type) {
            case ValueTypes.ImageTransformation:
                values[t.Key] = lib.execUnitToImage.get(t.ExecUnitName)!
                break
            case ValueTypes.ServiceAccountAnnotationTransformation:
                values[t.Key] = lib.execUnitToRole.get(t.ExecUnitName)!.arn
                break
            case ValueTypes.TargetGroupTransformation:
                values[t.Key] = eks.execUnitToTargetGroupArn.get(t.ExecUnitName)!
                break
            case ValueTypes.TargetGroupTransformation:
                values[t.Key] = eks.execUnitToTargetGroupArn.get(t.ExecUnitName)!
                break
            case ValueTypes.EnvironmentVariableTransformation:
                // Currently the only env vars we set are persist related
                // This will need to be changed to be more extensible
                values[t.Key] = lib.connectionString.get(t.Key)
                break
            default:
                throw new Error(`Unsupported Transformation Type ${t.Key}`)
        }
    })
    return values
}

interface applyChartParams {
    eks: Eks
    chartName: string
    values: Value[]
    dependsOn: any[]
    provider: pulumi_k8s.Provider
}

export const applyChart = (lib: CloudCCLib, args: applyChartParams) => {
    const values = getChartValues(lib, args.eks, args.values)
    new pulumi_k8s.helm.v3.Chart(
        `${args.eks.clusterName}-${args.chartName}`,
        {
            path: `./charts/${args.chartName}`,
            values,
        },
        { dependsOn: args.dependsOn, provider: args.provider }
    )
}
