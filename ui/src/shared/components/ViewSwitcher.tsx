// Libraries
import React, {FunctionComponent} from 'react'
import {Plot, FromFluxResult} from '@influxdata/giraffe'

// Components
import GaugeChart from 'src/shared/components/GaugeChart'
import SingleStat from 'src/shared/components/SingleStat'
import TableGraphs from 'src/shared/components/tables/TableGraphs'
import HistogramPlot from 'src/shared/components/HistogramPlot'
import HeatmapPlot from 'src/shared/components/HeatmapPlot'
import FluxTablesTransform from 'src/shared/components/FluxTablesTransform'
import XYPlot from 'src/shared/components/XYPlot'
import ScatterPlot from 'src/shared/components/ScatterPlot'
import LatestValueTransform from 'src/shared/components/LatestValueTransform'
import CheckPlot from 'src/shared/components/CheckPlot'

// Types
import {
  CheckViewProperties,
  QueryViewProperties,
  RemoteDataState,
  SingleStatViewProperties,
  StatusRow,
  TimeZone,
  XYViewProperties,
  TimeRange,
  CheckType,
  Threshold,
} from 'src/types'

interface Props {
  giraffeResult: FromFluxResult
  files: string[]
  loading: RemoteDataState
  properties: QueryViewProperties | CheckViewProperties
  timeZone: TimeZone
  statuses: StatusRow[][]
  timeRange: TimeRange | null
  checkType?: CheckType
  checkThresholds?: Threshold[]
}

const ViewSwitcher: FunctionComponent<Props> = ({
  properties,
  loading,
  timeRange,
  files,
  giraffeResult: {table, fluxGroupKeyUnion},
  timeZone,
  statuses,
  checkType = null,
  checkThresholds = [],
}) => {
  switch (properties.type) {
    case 'single-stat':
      return (
        <LatestValueTransform table={table} allowString={true}>
          {latestValue => (
            <SingleStat stat={latestValue} properties={properties} />
          )}
        </LatestValueTransform>
      )

    case 'table':
      return (
        <FluxTablesTransform files={files}>
          {tables => (
            <TableGraphs
              tables={tables}
              properties={properties}
              timeZone={timeZone}
            />
          )}
        </FluxTablesTransform>
      )

    case 'gauge':
      return (
        <LatestValueTransform table={table} allowString={false}>
          {latestValue => (
            <GaugeChart value={latestValue} properties={properties} />
          )}
        </LatestValueTransform>
      )
    case 'xy':
      return (
        <XYPlot
          timeRange={timeRange}
          fluxGroupKeyUnion={fluxGroupKeyUnion}
          loading={loading}
          table={table}
          timeZone={timeZone}
          viewProperties={properties}
        >
          {config => <Plot config={config} />}
        </XYPlot>
      )

    case 'line-plus-single-stat':
      const xyProperties = {
        ...properties,
        colors: properties.colors.filter(c => c.type === 'scale'),
        type: 'xy' as 'xy',
        geom: 'line' as 'line',
      } as XYViewProperties

      const singleStatProperties = {
        ...properties,
        tickPrefix: '',
        tickSuffix: '',
        colors: properties.colors.filter(c => c.type !== 'scale'),
        type: 'single-stat',
      } as SingleStatViewProperties

      return (
        <XYPlot
          timeRange={timeRange}
          fluxGroupKeyUnion={fluxGroupKeyUnion}
          loading={loading}
          table={table}
          timeZone={timeZone}
          viewProperties={xyProperties}
        >
          {config => (
            <Plot config={config}>
              <LatestValueTransform
                table={config.table}
                quiet={true}
                allowString={false}
              >
                {latestValue => (
                  <SingleStat
                    stat={latestValue}
                    properties={singleStatProperties}
                  />
                )}
              </LatestValueTransform>
            </Plot>
          )}
        </XYPlot>
      )

    case 'histogram':
      return (
        <HistogramPlot
          table={table}
          loading={loading}
          timeZone={timeZone}
          viewProperties={properties}
        >
          {config => <Plot config={config} />}
        </HistogramPlot>
      )

    case 'heatmap':
      return (
        <HeatmapPlot
          timeRange={timeRange}
          loading={loading}
          table={table}
          timeZone={timeZone}
          viewProperties={properties}
        >
          {config => <Plot config={config} />}
        </HeatmapPlot>
      )

    case 'scatter':
      return (
        <ScatterPlot
          timeRange={timeRange}
          loading={loading}
          table={table}
          viewProperties={properties}
          timeZone={timeZone}
        >
          {config => <Plot config={config} />}
        </ScatterPlot>
      )

    case 'check':
      return (
        <CheckPlot
          checkType={checkType}
          thresholds={checkThresholds}
          table={table}
          fluxGroupKeyUnion={fluxGroupKeyUnion}
          loading={loading}
          timeZone={timeZone}
          viewProperties={properties}
          statuses={statuses}
        >
          {config => <Plot config={config} />}
        </CheckPlot>
      )

    default:
      throw new Error('Unknown view type in <ViewSwitcher /> ')
  }
}

export default ViewSwitcher
